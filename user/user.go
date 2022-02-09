package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/buzkaaclicker/backend/discord"
	"github.com/buzkaaclicker/backend/profile"
	"github.com/uptrace/bun"
)

const LocalsKey = "user"

type Model struct {
	bun.BaseModel `bun:"table:user"`

	Id                  int64          `bun:",pk,autoincrement" json:"-"`
	CreatedAt           time.Time      `bun:",nullzero,notnull,default:current_timestamp" json:"-"`
	RolesNames          []RoleId       `bun:",notnull,array" json:"-"`
	DiscordId           string         `bun:",notnull,unique" json:"-"`
	DiscordRefreshToken string         `bun:",notnull" json:"-"`
	Email               string         `bun:"email,notnull" json:"-"`
	Profile             *profile.Model `bun:"rel:has-one,join:id=user_id"`

	// Mapped (in AfterScanRow hook) roles from RolesNames.
	Roles Roles `bun:"-"`
}

var _ bun.AfterScanRowHook = (*Model)(nil)

func (u *Model) AfterScanRow(ctx context.Context) error {
	u.Roles = make(Roles, 0, len(u.RolesNames))
	for _, n := range u.RolesNames {
		role, ok := AllRoles[n]
		if ok {
			u.Roles = append(u.Roles, role)
		}
	}
	return nil
}

type Store struct {
	DB *bun.DB
}

func (s *Store) RegisterDiscordUser(ctx context.Context,
	dcUser discord.User, refreshToken string) (*Model, error) {
	user := &Model{
		DiscordId:           dcUser.Id,
		DiscordRefreshToken: refreshToken,
		Email:               dcUser.Email,
		RolesNames:          []RoleId{},
	}

	err := s.DB.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(user).
			On(`CONFLICT (discord_id) DO UPDATE SET email=EXCLUDED.email, ` +
				`discord_refresh_token=EXCLUDED.discord_refresh_token`).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert user: %w", err)
		}

		profile := &profile.Model{
			UserId:    user.Id,
			Name:      dcUser.Username,
			AvatarUrl: dcUser.AvatarUrl(),
		}
		_, err = tx.NewInsert().
			Model(profile).
			On(`CONFLICT (user_id) DO UPDATE SET name=EXCLUDED.name, avatar_url=EXCLUDED.avatar_url`).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert profile: %w", err)
		}
		user.Profile = profile
		return nil
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Store) ById(ctx context.Context, userId int64) (*Model, error) {
	user := new(Model)
	err := s.DB.NewSelect().
		Model(user).
		Where(`"model"."id"=?`, userId).
		Relation("Profile").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("select user: %w", err)
	}
	return user, nil
}