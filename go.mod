module wgo

replace (
	github.com/minio/minio-go => ./vendor/github.com/minio/minio-go
	golang.org/x/crypto => ./vendor/golang.org/x/crypto
	golang.org/x/net => ./vendor/golang.org/x/net
	golang.org/x/sys => ./vendor/golang.org/x/sys
	golang.org/x/text => ./vendor/golang.org/x/text
	google.golang.org/genproto => ./vendor/google.golang.org/genproto
	google.golang.org/grpc => ./vendor/google.golang.org/grpc
    google.golang.org/appengine => ./vendor/google.golang.org/appengine
	gorp => ./vendor/gorp
)

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/bitly/go-simplejson v0.5.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dustin/randbo v0.0.0-20140428231429-7f1b564ca724
	github.com/fortytw2/leaktest v1.3.0 // indirect
	github.com/garyburd/redigo v1.6.0
	github.com/go-ini/ini v1.42.0 // indirect
	github.com/go-sql-driver/mysql v1.4.1
	github.com/golang/protobuf v1.3.1 // indirect
	github.com/google/go-cmp v0.2.0 // indirect
	github.com/klauspost/compress v1.4.1
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lib/pq v1.0.0 // indirect
	github.com/mailru/easyjson v0.0.0-20190312143242-1de009706dbe // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/minio/minio-go v0.0.0-00010101000000-000000000000
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d
	github.com/olivere/elastic v6.2.16+incompatible
	github.com/pkg/errors v0.8.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190306220146-200a235640ff // indirect
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/viper v1.3.2
	github.com/valyala/fasthttp v1.2.0
	github.com/ziutek/mymysql v1.5.4 // indirect
	golang.org/x/crypto v0.0.0-20181203042331-505ab145d0a9
	golang.org/x/net v0.0.0-20180911220305-26e67e76b6c3
	google.golang.org/genproto v0.0.0-00010101000000-000000000000 // indirect
	google.golang.org/grpc v0.0.0-00010101000000-000000000000
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/ini.v1 v1.42.0 // indirect
	gopkg.in/yaml.v2 v2.2.2
	gorp v0.0.0-00010101000000-000000000000
)
