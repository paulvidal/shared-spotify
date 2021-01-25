module github.com/shared-spotify

go 1.15

require (
	github.com/DataDog/datadog-go v4.2.0+incompatible
	github.com/aws/aws-sdk-go v1.35.14 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/golang-lru v0.5.1
	github.com/klauspost/compress v1.11.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/minchao/go-apple-music v0.0.0-20210121005645-8895087fad1d
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/philhofer/fwd v1.1.0 // indirect
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.7.0
	github.com/xdg/stringprep v1.0.0 // indirect
	github.com/zmb3/spotify v0.0.0-20200814173021-9bec46940cc0
	go.mongodb.org/mongo-driver v1.4.3
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/net v0.0.0-20200904194848-62affa334b73 // indirect
	golang.org/x/oauth2 v0.0.0-20200902213428-5d25da1a8d43
	golang.org/x/sys v0.0.0-20200905004654-be1d3432aa8f // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	gopkg.in/DataDog/dd-trace-go.v1 v1.27.1
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace github.com/minchao/go-apple-music => github.com/paulvidal/go-apple-music v0.0.0-20210124225748-d7f34a0138e2
