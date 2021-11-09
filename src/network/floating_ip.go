package network

import (
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models/network"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"strconv"
)

var FloatingIpSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "region",
		Required: true,
		Type: utils.String,
	},
	utils.Scheme{
		Param: "limit",
		Required: false,
		Type: utils.Int,
		Default: utils.DefaultLimit,
	},
	utils.Scheme{
		Param: "marker",
		Required: false,
		Type: utils.String,
		Default: "1",
	},
}

type FloatingIp struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (fip *FloatingIp)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	fip.credential = credential
	fip.client = cli
	return nil
}

func (FloatingIp)Call(params input.Params, replay *input.Replay) error {
	fip := &FloatingIp{}
	var next string
	var err error
	var fips []interface{}
	params.Args, err = utils.CheckParam(params.Args, SecurityGroupSchemes)
	if err != nil {
		return err
	}
	err = fip.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	region := params.Args["region"].(string)
	if next == "" {
		return nil
	}
	number, err := strconv.ParseInt(next,10,32)
	if err != nil {
		return errors.New("bad page number[marker] info")
	}
	pageNum := int32(number)
	query := map[string]interface{} {
		"RegionId": region,
		"CloudType": common.CloudType,
		"AccountId": fip.credential.AccountId,
	}
	request := &ecs20140526.DescribeEipAddressesRequest{
		RegionId: &region,
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	resp, err := fip.client.DescribeEipAddresses(request)
	if err != nil {
		return err
	}
	if resp.Body.EipAddresses == nil || resp.Body.EipAddresses.EipAddress == nil {
		return errors.New("bad response for query vSwitch")
	}
	for _, b := range resp.Body.EipAddresses.EipAddress {
		s := network.NewFloatingIpModel()
		s.Deleted = 0
		s.CloudType = common.CloudType
		s.AccountId = fip.credential.AccountId
		s.RegionId = region
		s.IpAddr = utils.SafeString(b.IpAddress)
		s.ProviderId = utils.SafeString(b.AllocationId)
		s.CreateTime = utils.SafeString(b.AllocationTime)
		s.ExpiredTime = utils.SafeString(b.ExpiredTime)
		s.Bandwidth = utils.SafeString(b.Bandwidth)
		s.BindResourceId = utils.SafeString(b.InstanceId)
		s.BindResourceType = utils.SafeString(b.InstanceType)
		s.Extra = map[string]interface{}{
			"ChargeType": utils.SafeString(b.ChargeType),
		}
		s.SetIndex()
		s.SetChecksum()
		checked, key := s.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		fips = append(fips, s)
	}
	if !utils.CheckQueryKeys(query, network.SecurityGroupModel{}) {
		return errors.New("query key is not attribute of FIpModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = fips
	return nil
}
