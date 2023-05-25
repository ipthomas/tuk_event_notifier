package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/ipthomas/tukcnst"
	"github.com/ipthomas/tukdbint"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var initSrvcs = false

func main() {
	lambda.Start(Handle_Request)
}
func Handle_Request(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	log.SetFlags(log.Lshortfile)
	var rsp []byte

	var err error
	var dbconn tukdbint.TukDBConnection
	if !initSrvcs {
		dbconn = tukdbint.TukDBConnection{DBUser: os.Getenv(tukcnst.ENV_DB_USER), DBPassword: os.Getenv(tukcnst.ENV_DB_PASSWORD), DBHost: os.Getenv(tukcnst.ENV_DB_HOST), DBPort: os.Getenv(tukcnst.ENV_DB_PORT), DBName: os.Getenv(tukcnst.ENV_DB_NAME)}
		if err = tukdbint.NewDBEvent(&dbconn); err != nil {
			log.Println(err.Error())
			return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
		}
		initSrvcs = true
	}
	log.Printf("Processing API Gateway %s Request", req.HTTPMethod)
	subs := tukdbint.Subscriptions{Action: tukcnst.SELECT}
	sub := tukdbint.Subscription{Pathway: req.QueryStringParameters[tukcnst.TUK_EVENT_QUERY_PARAM_PATHWAY], Topic: req.QueryStringParameters["email"], NhsId: req.QueryStringParameters[tukcnst.TUK_EVENT_QUERY_PARAM_NHS]}
	subs.Subscriptions = append(subs.Subscriptions, sub)
	if err = tukdbint.NewDBEvent(&subs); err != nil {
		log.Println(err.Error())
		return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
	}
	var sendto []string
	for _, v := range subs.Subscriptions {
		if (v.Expression == "" || v.Expression == sub.Expression) && (v.NhsId == "" || v.NhsId == sub.NhsId) {
			sendto = append(sendto, v.BrokerRef)
		}
	}
	rsp, _ = json.Marshal(sendto)
	return queryResponse(http.StatusOK, string(rsp), tukcnst.APPLICATION_JSON)
}

func setAwsResponseHeaders(contentType string) map[string]string {
	awsHeaders := make(map[string]string)
	awsHeaders["Server"] = "Event_Notifier"
	awsHeaders["Access-Control-Allow-Origin"] = "*"
	awsHeaders["Access-Control-Allow-Headers"] = "accept, Content-Type"
	awsHeaders["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	awsHeaders[tukcnst.CONTENT_TYPE] = contentType
	return awsHeaders
}
func queryResponse(statusCode int, body string, contentType string) (*events.APIGatewayProxyResponse, error) {
	log.Println(body)
	return &events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    setAwsResponseHeaders(contentType),
		Body:       body,
	}, nil
}
