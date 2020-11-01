# duct: a Golang integration testing helper for Docker

duct uses structs similar in a fashion to the way `docker-compose` uses YAML to
launch your containers. This is so much faster and easier to control; why shell
out to `docker-compose` at all?

Check out the [godoc](https://pkg.go.dev/github.com/erikh/duct).

## Example

### Running Containers

Here's how you might launch [Gitea](https://gitea.io) in duct:

```go
package main

import (
  "context"
  "log"
  "net"
  "testing"
  "time"

  "github.com/erikh/duct"
  dc "github.com/fsouza/go-dockerclient"
)

func TestStartGitea(t *testing.T) {
  c := duct.New(duct.Manifest{
    {
      Name: "gitea-postgres",
      Env: []string{
        "POSTGRES_USER=gitea",
        "POSTGRES_PASSWORD=gitea",
        "POSTGRES_DB=gitea",
      },
      Image:    "postgres:latest",
      BootWait: 2 * time.Second,
    },
    {
      Name: "gitea",
      Env: []string{
        "USER_UID=1000",
        "USER_GID=1000",
        "DB_TYPE=postgres",
        "DB_HOST=gitea-postgres:5432",
        "DB_USER=gitea",
        "DB_NAME=gitea",
        "DB_PASSWD=gitea",
        "DOMAIN=gitea",
        "ROOT_URL=http://gitea:11498",
        "DISABLE_SSH=true",
        "OAUTH2_ENABLE=true",
        "OAUTH2_JWT_SECRET=mysecret",
        "INSTALL_LOCK=true",
      },
      Image:    "gitea/gitea:1.12",
      BootWait: 2 * time.Second,
      AliveFunc: func(ctx context.Context, client *dc.Client, id string) error {
        for {
          conn, err := net.Dial("tcp", "localhost:11498")
          if err != nil {
            log.Printf("Error while dialing container: %v", err)
            time.Sleep(100 * time.Millisecond)
            continue
          }
          conn.Close()
          return nil
        }
      },
      PostCommands: [][]string{
        {
          "gitea", "admin", "create-user",
          "--username", "erikh",
          "--password", "erikh",
          "--email", "erikh@example.org",
        },
      },
      PortForwards: map[int]int{
        11498: 3000,
      },
    },
  }, "gitea-integration-test")

  // Ctrl+C and SIGTERM will tear this down, and pass it up to the test suite
  c.HandleSignals(true)

  t.Cleanup(func() {
    if err := c.Teardown(context.Background()); err != nil {
      t.Fatal(err)
    }
  })

  if err := c.Launch(context.Background()); err != nil {
    t.Fatal(err)
  }

  // do something with gitea
}
```

### Builder support

duct has very basic builder and context support. Make sure to use the
`LocalImage` flag when using these images in your container manifests so they
don't get pulled. Builds are logged to stderr in a very similar fashion to
`docker build`.

```go
b := Builder{
  "test-image": {
    Dockerfile: "testdata/Dockerfile.test",
    Context:    ".",
  },
  "test-image2": {
    Dockerfile: "testdata/Dockerfile.test",
  },
}

if err := b.Run(context.Background()); err != nil {
  t.Fatal(err)
}

c := New(Manifest{
  {
    Name:       "test-image",
    Image:      "test-image",
    LocalImage: true,
  },
}, "duct-test-network")

if err := c.Launch(context.Background()); err != nil {
  t.Fatal(err)
}

if err := c.Teardown(context.Background()); err != nil {
  t.Fatal(err)
}
```

### Example Log Output

duct has nice logging so you can figure out what the heck is going on. From the code above:

```
=== RUN   TestStartGitea
2020/10/31 23:38:35 Pulling docker image: [postgres:latest]
2020/10/31 23:38:37 Creating container: [gitea-postgres]
2020/10/31 23:38:37 Pulling docker image: [gitea/gitea:1.12]
2020/10/31 23:38:38 Creating container: [gitea]
2020/10/31 23:38:39 Starting container: [gitea-postgres]
2020/10/31 23:38:39 Sleeping for 2s (requested by "gitea-postgres" bootWait parameter)
2020/10/31 23:38:41 Starting container: [gitea]
2020/10/31 23:38:41 Sleeping for 2s (requested by "gitea" bootWait parameter)
2020/10/31 23:38:43 Running aliveFunc for gitea
2020/10/31 23:38:43 AliveFunc for gitea completed
2020/10/31 23:38:43 Running post-command [gitea admin create-user --username erikh --password erikh --email erikh@example.org] in container: [gitea]
2020/11/01 06:38:43 ...dules/setting/git.go:93:newGit() [I] Git Version: 2.24.3, Wire Protocol Version 2 Enabled
2020/11/01 06:38:43 ...m.io/xorm/core/db.go:154:QueryContext() [I] [SQL] SELECT count(*) FROM "user" WHERE (type=0) [] - 5.089469ms
2020/11/01 06:38:43 ...m.io/xorm/core/tx.go:36:BeginTx() [I] [SQL] BEGIN TRANSACTION [] - 146.59µs
2020/11/01 06:38:43 ...m.io/xorm/core/tx.go:157:QueryContext() [I] [SQL] SELECT "id", "lower_name", "name", "full_name", "email", "keep_email_private", "email_notifications_preference", "passwd", "passwd_hash_algo", "must_change_password", "login_type", "login_source", "login_name", "type", "location", "website", "rands", "salt", "language", "description", "created_unix", "updated_unix", "last_login_unix", "last_repo_visibility", "max_repo_creation", "is_active", "is_admin", "is_restricted", "allow_git_hook", "allow_import_local", "allow_create_organization", "prohibit_login", "avatar", "avatar_email", "use_custom_avatar", "num_followers", "num_following", "num_stars", "num_repos", "num_teams", "num_members", "visibility", "repo_admin_change_team_access", "diff_view_style", "theme" FROM "user" WHERE (id!=$1) AND "lower_name"=$2 LIMIT 1 [0 erikh] - 1.26592ms
2020/11/01 06:38:43 ...m.io/xorm/core/tx.go:157:QueryContext() [I] [SQL] SELECT "id", "lower_name", "name", "full_name", "email", "keep_email_private", "email_notifications_preference", "passwd", "passwd_hash_algo", "must_change_password", "login_type", "login_source", "login_name", "type", "location", "website", "rands", "salt", "language", "description", "created_unix", "updated_unix", "last_login_unix", "last_repo_visibility", "max_repo_creation", "is_active", "is_admin", "is_restricted", "allow_git_hook", "allow_import_local", "allow_create_organization", "prohibit_login", "avatar", "avatar_email", "use_custom_avatar", "num_followers", "num_following", "num_stars", "num_repos", "num_teams", "num_members", "visibility", "repo_admin_change_team_access", "diff_view_style", "theme" FROM "user" WHERE (email=$1) LIMIT 1 [erikh@example.org] - 668.11µs
2020/11/01 06:38:43 ...m.io/xorm/core/tx.go:157:QueryContext() [I] [SQL] SELECT "id", "uid", "email", "is_activated" FROM "email_address" WHERE (email=$1) LIMIT 1 [erikh@example.org] - 658.433µs
2020/11/01 06:38:43 ...m.io/xorm/core/tx.go:157:QueryContext() [I] [SQL] INSERT INTO "user" ("lower_name","name","full_name","email","keep_email_private","email_notifications_preference","passwd","passwd_hash_algo","must_change_password","login_type","login_source","login_name","type","location","website","rands","salt","language","description","created_unix","updated_unix","last_login_unix","last_repo_visibility","max_repo_creation","is_active","is_admin","is_restricted","allow_git_hook","allow_import_local","allow_create_organization","prohibit_login","avatar","avatar_email","use_custom_avatar","num_followers","num_following","num_stars","num_repos","num_teams","num_members","visibility","repo_admin_change_team_access","diff_view_style","theme") VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39, $40, $41, $42, $43, $44) RETURNING "id" [erikh erikh  erikh@example.org false enabled c9448c0f954aaa240ba79f1d77c38a167f83479af7a8509a0f52affa28a937dbd2edbe6b3d63ac551567e76659f2a42b410c pbkdf2 false 0 0  0   CNBdZZp36u BUK9GUe8Ih   1604212723 1604212723 0 false -1 true false false false false false false 7c26bac970c7b33ad8f3e5a905d82a0c erikh@example.org false 0 0 0 0 0 0 public false  gitea] - 1.112965ms
New user 'erikh' has been successfully created!
2020/10/31 23:38:43 Killing container: [gitea-postgres]
2020/10/31 23:38:44 Removing container: [gitea-postgres]
2020/10/31 23:38:44 Killing container: [gitea]
2020/10/31 23:38:44 Removing container: [gitea]
--- PASS: TestStartGitea (9.36s)
PASS
ok      github.com/erikh/tmp    9.364s
```

## Roadmap:

- [ ] Better `*testing.T` integrations with e.g., Cleanup directly
- [ ] Stdio handling and sniffing
- [ ] Attach handling

# Author

Erik Hollensbe <github@hollensbe.org>
