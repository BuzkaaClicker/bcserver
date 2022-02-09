package program

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/buzkaaclicker/backend/pgdb"
	"github.com/buzkaaclicker/backend/rest"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestDownloadProgram(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
		return
	}
	assert := assert.New(t)
	ctx := context.Background()

	db := pgdb.OpenTest(ctx)
	defer db.Close()

	app := fiber.New(fiber.Config{ErrorHandler: rest.ErrorHandler})
	controller := Controller{
		Repo: PgRepo{DB: db},
	}
	controller.InstallTo(app)

	exampleFile := []File{{Path: "installer.pkg", DownloadUrl: "https://buzkaaclicker.pl/sample", Hash: "256"}}
	_, err := db.NewInsert().Model(&[]Model{
		{Type: "installer", OS: "macOS", Arch: "x86-64", Branch: "stable",
			Files: []File{{Path: "installer.pkg", DownloadUrl: "https://buzkaaclicker.pl/sample", Hash: "499"}}},
		{Type: "installer", OS: "macOS", Arch: "x86-64", Branch: "beta", Files: exampleFile},
		{Type: "installer", OS: "macOS", Arch: "arm64", Branch: "stable", Files: exampleFile},
		{Type: "installer", OS: "Windows", Arch: "x86-64", Branch: "stable", Files: exampleFile},
		{Type: "installer", OS: "Windows", Arch: "arm8", Branch: "alpha", Files: exampleFile},
		{Type: "clicker", OS: "macOS", Arch: "x86-64", Branch: "stable",
			Files: []File{{Path: "installer.pkg", DownloadUrl: "https://buzkaaclicker.pl/sample", Hash: "1"}}},
	}).Exec(ctx)
	assert.NoError(err)

	cases := []struct {
		url  string
		body string
	}{
		{"/download/installer?os=macOS&arch=x86-64&branch=stable",
			`[{"path":"installer.pkg","download_url":"https://buzkaaclicker.pl/sample","hash":"499"}]`},
		{"/download/clicker?os=macOS&arch=x86-64&branch=stable",
			`[{"path":"installer.pkg","download_url":"https://buzkaaclicker.pl/sample","hash":"1"}]`},
		{"/download/clicker?os=macOS&arch=arm64&branch=stable", `{"error_message":"Not Found"}`},
		{"/download/clicker?os=macOS&arch=x86-64&branch=unstable", `{"error_message":"Not Found"}`},
		{"/download/clicker?os=macOSes&arch=x86-64&branch=stable", `{"error_message":"Not Found"}`},
		{"/download/clicker?os=Windows&arch=x86-64&branch=stable", `{"error_message":"Not Found"}`},
		{"/download/installer?os=Windows&arch=x86-64&branch=stable",
			`[{"path":"installer.pkg","download_url":"https://buzkaaclicker.pl/sample","hash":"256"}]`},
	}

	for _, tc := range cases {
		req := httptest.NewRequest("GET", tc.url, nil)
		resp, err := app.Test(req)
		assert.NoError(err)
		defer resp.Body.Close()

		assert.Equal(resp.Header.Get("Content-Type"), fiber.MIMEApplicationJSON, "Invalid content type")
		body, err := ioutil.ReadAll(resp.Body)
		assert.NoError(err)
		assert.Equal(tc.body, string(body), "Response body not equal")
	}
}