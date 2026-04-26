module github.com/donnie-ellis/aop/controller

go 1.25.8

replace github.com/donnie-ellis/aop/pkg => ../pkg

require (
	github.com/donnie-ellis/aop/pkg v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.9.2
	github.com/robfig/cron/v3 v3.0.1
	github.com/rs/zerolog v1.35.1
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
