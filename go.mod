module github/kubernetes-sigs/ingate

go 1.24.1

require github.com/spf13/cobra v1.9.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
)

replace github.com/kubernetes-sigs/ingate/internal/cmd/version => ./internal/cmd/version

replace github.com/kubernetes-sigs/ingate/cmd/ingate => ./cmd/ingate

replace github.com/kubernetes-sigs/ingate/cmd/ingate/root => ./cmd/root
