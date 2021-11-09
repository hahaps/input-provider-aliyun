package compute

import (
	"encoding/json"
	"errors"
	"fmt"
	cms20190101 "github.com/alibabacloud-go/cms-20190101/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models"
	"github.com/hahaps/common-provider/src/models/compute"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"strconv"
	"strings"
	"time"
)

var ServerMetricNamespace string = "acs_ecs_dashboard"

var MetricMap = map[string]map[string]string{
	"AdvanceCredit": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Count",
		"EDimensions": "",
	},
	"BurstCredit": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Count",
		"EDimensions": "",
	},
	"CPUUtilization": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "%",
		"EDimensions": "",
	},
	"DiskReadBPS": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Byte/s",
		"EDimensions": "",
	},
	"DiskReadIOPS": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Count/Second",
		"EDimensions": "",
	},
	"DiskWriteBPS": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Byte/s",
		"EDimensions": "",
	},
	"DiskWriteIOPS": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Count/Second",
		"EDimensions": "",
	},
	"InternetIn": map[string]string{
		"Statistics": "Average, Minimum, Maximum, Sum",
		"Unit": "Byte",
		"EDimensions": "",
	},
	"InternetInRate": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "bit/s",
		"EDimensions": "",
	},
	"InternetOut": map[string]string{
		"Statistics": "Average, Minimum, Maximum, Sum",
		"Unit": "Byte",
		"EDimensions": "",
	},
	"InternetOutRate": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "bit/s",
		"EDimensions": "",
	},
	"InternetOutRate_Percent": map[string]string{
		"Statistics": "Average",
		"Unit": "%",
		"EDimensions": "",
	},
	"IntranetIn": map[string]string{
		"Statistics": "Average, Minimum, Maximum, Sum",
		"Unit": "Byte",
		"EDimensions": "",
	},
	"IntranetInRate": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "bit/s",
		"EDimensions": "",
	},
	"IntranetOut": map[string]string{
		"Statistics": "Average, Minimum, Maximum, Sum",
		"Unit": "Byte",
		"EDimensions": "",
	},
	"IntranetOutRate": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "bit/s",
		"EDimensions": "",
	},
	"NotpaidSurplusCredit": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Count",
		"EDimensions": "",
	},
	"TotalCredit": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "Count",
		"EDimensions": "",
	},
	"VPC_PublicIP_InternetInRate": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "bit/s",
		"EDimensions": "ip",
	},
	"VPC_PublicIP_InternetOutRate": map[string]string{
		"Statistics": "Average, Minimum, Maximum",
		"Unit": "bit/s",
		"EDimensions": "ip",
	},
	"VPC_PublicIP_InternetOutRate_Percent": map[string]string{
		"Statistics": "Average",
		"Unit": "%",
		"EDimensions": "ip",
	},
	"aep_bw_read": map[string]string{
		"Statistics": "Average",
		"Unit": "MB/s",
		"EDimensions": "aepName",
	},
	"aep_bw_write": map[string]string{
		"Statistics": "Average",
		"Unit": "MB/s",
		"EDimensions": "aepName",
	},
	"aep_iops_read": map[string]string{
		"Statistics": "Average",
		"Unit": "Count",
		"EDimensions": "aepName",
	},
	"aep_iops_write": map[string]string{
		"Statistics": "Average",
		"Unit": "Count",
		"EDimensions": "aepName",
	},
	"concurrentConnections": map[string]string{
		"Statistics": "Maximum",
		"Unit": "Count",
		"EDimensions": "",
	},
	"eip_InternetInRate": map[string]string{
		"Statistics": "Value",
		"Unit": "bit/s",
		"EDimensions": "",
	},
	"eip_InternetOutRate": map[string]string{
		"Statistics": "Value",
		"Unit": "bit/s",
		"EDimensions": "",
	},
}

var ServerMetricSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "region",
		Required: true,
		Type: utils.String,
	},
	utils.Scheme{
		Param: "metric_names",
		Required: true,
		Type: utils.Slice,
	},
	utils.Scheme{
		Param: "period",
		Required: false,
		Type: utils.Int,
		Default: 60,
	},
}

type ServerMetric struct {
	client *cms20190101.Client
	credential input.Credential
	input.Resource
}

func (m *ServerMetric)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("metrics.cn-hangzhou.aliyuncs.com")
	cli, err := cms20190101.NewClient(config)
	if err != nil {
		return err
	}
	m.credential = credential
	m.client = cli
	return nil
}

func (ServerMetric)Call(params input.Params, replay *input.Replay) (err error) {
	params.Args, err = utils.CheckParam(params.Args, ServerMetricSchemes)
	if err != nil {
		return err
	}
	metricNames := params.Args["metric_names"].([]interface{})
	for _, mn := range metricNames {
		metricName := strings.TrimSpace(fmt.Sprint(mn))
		if _, ok := MetricMap[metricName]; !ok {
			return errors.New("bad metric " + metricName)
		}
	}
	err = Server{}.Call(params, replay)
	if err != nil {
		return err
	}
	if len(replay.Result) == 0 {
		return err
	}
	metric := &ServerMetric{}
	err = metric.init(params.Credential)
	if err != nil {
		return err
	}
	cpm := map[string]*compute.ServerModel{}
	dimensions := "["
	for _, ser := range replay.Result {
		sv := ser.(*compute.ServerModel)
		cpm[sv.ProviderId] = sv
		dimensions += "{instanceId: " + sv.ProviderId + "}, "
	}
	dimensions += "]"
	region := params.Args["region"].(string)
	period := strconv.Itoa(params.Args["period"].(int))
	timestamp := time.Now().Unix()
	query := map[string]interface{} {
		"CloudType": common.CloudType,
		"AccountId": metric.credential.AccountId,
		"Index": strconv.FormatInt(timestamp, 10),
	}
	var metrs []interface{}
	for _, mn := range metricNames {
		metricName := strings.TrimSpace(fmt.Sprint(mn))
		request := &cms20190101.DescribeMetricLastRequest{
			Period: &period,
			Namespace: &ServerMetricNamespace,
			MetricName: &metricName,
			Dimensions: &dimensions,
		}
		resp, err := metric.client.DescribeMetricLast(request)
		if err != nil {
			return err
		}
		if resp.Body.Success == nil || (resp.Body.Success != nil && !(*resp.Body.Success)) {
			return errors.New(utils.SafeString(resp.Body.Message))
		}
		var dataPoints []map[string]interface{}
		err = json.Unmarshal([]byte(utils.SafeString(resp.Body.Datapoints)), &dataPoints)
		if err != nil {
			return err
		}
		statistics := strings.Split(MetricMap[metricName]["Statistics"], ", ")
		for _, b := range dataPoints {
			for _, stat := range statistics {
				metr := models.NewMetricModel()
				metr.Deleted = 0
				metr.CloudType = common.CloudType
				metr.AccountId = metric.credential.AccountId
				metr.InstanceId = b["instanceId"].(string)
				metr.Value = fmt.Sprint(b[stat])
				metr.Unit = MetricMap[metricName]["Unit"]
				metr.MetricTime = b["timestamp"].(float64)
				metr.Name = metricName + "." + stat
				if MetricMap[metricName]["EDimensions"] != "" {
					metr.Name += "/" + MetricMap[metricName]["EDimensions"]
				}
				metr.Extra = map[string]interface{}{
					"Region": region,
					"InstanceName": cpm[metr.InstanceId].Name,
					"PrimaryNicIp": cpm[metr.InstanceId].PrimaryNicIp,
					"PrimaryNicFloatingIp": cpm[metr.InstanceId].PrimaryNicFloatingIp,
				}
				checked, key := metr.CheckRequired()
				if !checked {
					return errors.New(
						fmt.Sprintf("Value[%v] should not be empty", key))
				}
				metr.SetIndex()
				metrs = append(metrs, &metr)
			}
		}
	}

	if !utils.CheckQueryKeys(query, models.MetricModel{}) {
		return errors.New("query key is not attribute of MetricModel")
	}
	replay.Query = query
	replay.Result = metrs
	return nil
}
