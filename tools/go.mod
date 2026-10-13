module github.com/dshills/elab/tools

go 1.26.0

require (
	github.com/BurntSushi/toml v1.6.0
	golang.org/x/exp/typeparams v0.0.0-20260811152304-ee035b5b010f
	golang.org/x/tools v0.50.0
	honnef.co/go/tools v0.8.1
)

require (
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/telemetry v0.0.0-20260908163034-4bcc4b2ee518 // indirect
)

tool (
	golang.org/x/tools/cmd/goimports
	honnef.co/go/tools/cmd/staticcheck
)
