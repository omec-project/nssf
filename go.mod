module github.com/free5gc/nssf

go 1.14

require (
	github.com/antonfisher/nested-logrus-formatter v1.3.0
	github.com/badhrinathpa/nssf v0.0.0-00010101000000-000000000000
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/free5gc/http2_util v1.0.0
	github.com/free5gc/http_wrapper v1.0.0
	github.com/free5gc/logger_conf v1.0.0
	github.com/free5gc/logger_util v1.0.0
	github.com/free5gc/openapi v1.0.0
	github.com/free5gc/path_util v1.0.0
	github.com/free5gc/version v1.0.0
	github.com/gin-gonic/gin v1.6.3
	github.com/google/uuid v1.1.2
	github.com/sirupsen/logrus v1.7.0
	github.com/urfave/cli v1.22.5
	google.golang.org/grpc v1.31.0
	google.golang.org/protobuf v1.25.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/badhrinathpa/nssf => ../nssf
