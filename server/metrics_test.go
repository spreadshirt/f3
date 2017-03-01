package server

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

type CloudwatchMock struct {
	cloudwatchiface.CloudWatchAPI
}

func (c *CloudwatchMock) PutMetricData(input *cloudwatch.PutMetricDataInput) (*cloudwatch.PutMetricDataOutput, error) {
	err := input.Validate()
	if err != nil {
		return nil, err
	}
	return &cloudwatch.PutMetricDataOutput{}, nil
}

func TestCloudwatchSender(t *testing.T) {
	cw := CloudwatchSender{
		hostname: "test-sender",
		metrics:  &CloudwatchMock{},
	}
	err := cw.SendGet(int64(21), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	err = cw.SendPut(int64(42), time.Now())
	if err != nil {
		t.Fatal(err)
	}
}
