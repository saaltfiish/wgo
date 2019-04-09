module wgo

go 1.12

replace (
	github.com/minio/minio-go => ./vendor/github.com/minio/minio-go
	github.com/valyala/fasthttp => ./vendor/github.com/valyala/fasthttp
	golang.org/x/net/lex/httplex => ./vendor/golang.org/x/net/lex/httplex
)

require (
	github.com/bitly/go-simplejson v0.5.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dustin/randbo v0.0.0-20140428231429-7f1b564ca724
	github.com/garyburd/redigo v1.6.0
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/klauspost/compress v1.4.1
	github.com/klauspost/cpuid v1.2.0 // indirect
	github.com/mailru/easyjson v0.0.0-20190312143242-1de009706dbe // indirect
	github.com/minio/minio-go v0.0.0-00010101000000-000000000000
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/olivere/elastic v6.2.17+incompatible
	github.com/pkg/errors v0.8.1 // indirect
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/viper v1.3.2
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.0.0-20190325154230-a5d413f7728c
	golang.org/x/net v0.0.0-20190328230028-74de082e2cca
	golang.org/x/net/lex/httplex v0.0.0-00010101000000-000000000000 // indirect
	google.golang.org/grpc v1.19.1
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/yaml.v2 v2.2.2
    github.com/stripe/stripe-go v53.1.0+incompatible
)
