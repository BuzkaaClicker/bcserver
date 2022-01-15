package main

import (
	"context"
	"testing"

	"github.com/buzkaaclicker/backend/discord"
	"github.com/stretchr/testify/assert"
)

func TestUserRoles(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}
	assert := assert.New(t)
	ctx := context.Background()

	app := createTestApp()

	_, err := app.db.NewCreateTable().
		IfNotExists().
		Model((*User)(nil)).
		Exec(ctx)
	assert.NoError(err)

	_, err = app.db.NewInsert().
		Model(&User{
			DiscordId:           "123",
			DiscordRefreshToken: "123",
			Email:               "user@rol.es",
			RolesNames:          []RoleId{RoleIdPro, RoleId("UNDEFINED role")},
		}).
		Exec(ctx)
	assert.NoError(err)

	var user User
	err = app.db.NewSelect().
		Model((*User)(nil)).
		Where("email=?", "user@rol.es").
		Scan(ctx, &user)
	assert.NoError(err)

	roles := user.Roles
	assert.Equal(Roles{AllRoles[RoleIdPro]}, roles)
	assert.Equal(AccessAllowed, roles.Access(PermissionDownloadPro))
	assert.Equal(AccessUndefined, roles.Access(PermissionAdminDashboard))
}

func TestRegisterDiscordUser(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}
	assert := assert.New(t)
	ctx := context.Background()

	app := createTestApp()

	discordUser := discord.User{
		Id:         "snowflake",
		Username:   "www_makin_cc",
		Email:      "clickacz@discord.makin.cc",
		AvatarHash: "f2789ef0ddaee56d91a782fa530b0009",
	}
	refreshToken := "21gokpoasio57"
	user, err := app.userStore.RegisterDiscordUser(ctx, discordUser, refreshToken)
	if !assert.NoError(err) {
		return
	}
	assert.Equal(discordUser.Id, user.DiscordId)
	assert.Equal(refreshToken, user.DiscordRefreshToken)
	assert.Equal(discordUser.Email, user.Email)

	userSel, err := app.userStore.ById(ctx, user.Id)
	if !assert.NoError(err) {
		return
	}
	assert.Equal(user, userSel)

	assert.Equal(discordUser.AvatarUrl(), userSel.Profile.AvatarUrl)
	assert.Equal(discordUser.Username, userSel.Profile.Name)
}
