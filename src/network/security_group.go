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

var SecurityGroupSchemes = []utils.Scheme {
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

type SecurityGroup struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (sg *SecurityGroup)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	sg.credential = credential
	sg.client = cli
	return nil
}

func (SecurityGroup)Call(params input.Params, replay *input.Replay) error {
	sg := &SecurityGroup{}
	var next string
	var err error
	var nics []interface{}
	params.Args, err = utils.CheckParam(params.Args, SecurityGroupSchemes)
	if err != nil {
		return err
	}
	err = sg.init(params.Credential)
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
		"AccountId": sg.credential.AccountId,
	}
	request := &ecs20140526.DescribeSecurityGroupsRequest{
		RegionId: &region,
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	resp, err := sg.client.DescribeSecurityGroups(request)
	if err != nil {
		return err
	}
	if resp.Body.SecurityGroups == nil || resp.Body.SecurityGroups.SecurityGroup == nil {
		return errors.New("bad response for query vSwitch")
	}
	for _, b := range resp.Body.SecurityGroups.SecurityGroup {
		s := network.NewSecurityGroupModel()
		s.Deleted = 0
		s.CloudType = common.CloudType
		s.AccountId = sg.credential.AccountId
		s.RegionId = region
		s.Name = utils.SafeString(b.SecurityGroupName)
		s.ProviderId = utils.SafeString(b.SecurityGroupId)
		s.CreateTime = utils.SafeString(b.CreationTime)
		s.Description = utils.SafeString(b.Description)
		s.Tags = getSGTags(b.Tags)
		s.Extra = map[string]interface{}{
			"ResourceGroupId": utils.SafeString(b.ResourceGroupId),
			"EcsCount": utils.SafeInt32(b.EcsCount),
			"VpcId": utils.SafeString(b.VpcId),
			"SecurityGroupType": utils.SafeString(b.SecurityGroupType),
		}
		s.SetIndex()
		s.SetChecksum()
		checked, key := s.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		nics = append(nics, s)
	}
	if !utils.CheckQueryKeys(query, network.SecurityGroupModel{}) {
		return errors.New("query key is not attribute of SecurityGroupModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = nics
	return nil
}

func getSGTags(tags *ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroupTags) string {
	if tags == nil {
		return ""
	}
	tgs := ""
	l := len(tags.Tag)
	for i, tg := range tags.Tag {
		tgs += fmt.Sprintf("%v=%v", tg.TagKey, tg.TagValue)
		if i + 1 < l {
			tgs += ";"
		}
	}
	return tgs
}
