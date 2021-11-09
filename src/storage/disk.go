package storage

import (
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models/compute"
	"github.com/hahaps/common-provider/src/models/storage"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"strconv"
)

var DiskSchemes = []utils.Scheme {
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

type Disk struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (disk *Disk)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs-cn-hangzhou.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	disk.credential = credential
	disk.client = cli
	return nil
}

func (Disk)Call(params input.Params, replay *input.Replay) error {
	disk := &Disk{}
	var next string
	var err error
	var disks []interface{}
	params.Args, err = utils.CheckParam(params.Args, DiskSchemes)
	if err != nil {
		return err
	}
	err = disk.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	region := params.Args["region"].(string)
	query := map[string]interface{} {
		"RegionId": region,
		"CloudType": common.CloudType,
		"AccountId": disk.credential.AccountId,
	}
	if next == "" {
		return nil
	}
	number, err := strconv.ParseInt(next,10,32)
	if err != nil {
		return errors.New("bad page number[marker] info")
	}
	pageNum := int32(number)
	request := &ecs20140526.DescribeDisksRequest{
		RegionId: tea.String(region),
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	resp, err := disk.client.DescribeDisks(request)
	if err != nil {
		return err
	}
	for _, dk := range resp.Body.Disks.Disk {
		dis := storage.NewDiskModel()
		dis.Deleted = 0
		dis.CloudType = common.CloudType
		dis.AccountId = disk.credential.AccountId
		dis.RegionId = region
		dis.ProviderId = utils.SafeString(dk.DiskId)
		dis.Name = utils.SafeString(dk.DiskName)
		dis.CreateTime = utils.SafeString(dk.CreationTime)
		dis.Status = utils.SafeString(dk.Status)
		dis.Size = utils.SafeInt32(dk.Size)
		dis.Type = utils.SafeString(dk.Type)
		dis.AttachedServer = utils.SafeString(dk.InstanceId)
		dis.Category = utils.SafeString(dk.Category)
		dis.ExpiredTime = utils.SafeString(dk.ExpiredTime)
		dis.Description = utils.SafeString(dk.Description)
		dis.Tags = getDiskTags(dk.Tags)
		dis.Attachments = getAttachments(dk.Attachments)
		dis.Extra = map[string]interface{}{
			"ResourceGroupId": utils.SafeString(dk.ResourceGroupId),
			"Encrypted": utils.SafeBool(dk.Encrypted, false),
			"DeleteAutoSnapshot": utils.SafeBool(dk.DeleteAutoSnapshot, false),
			"MultiAttach": utils.SafeString(dk.MultiAttach),
			"ImageId": utils.SafeString(dk.ImageId),
			"ZoneId": utils.SafeString(dk.ZoneId),
			"Device": utils.SafeString(dk.Device),
			"SourceSnapshotId": utils.SafeString(dk.SourceSnapshotId),
		}
		dis.SetIndex()
		dis.SetChecksum()
		checked, key := dis.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		disks = append(disks, dis)
	}
	if !utils.CheckQueryKeys(query, compute.ServerModel{}) {
		return errors.New("query key is not attribute of DiskModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = disks
	return nil
}

func getDiskTags(tags *ecs20140526.DescribeDisksResponseBodyDisksDiskTags) string {
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

func getAttachments(attach *ecs20140526.DescribeDisksResponseBodyDisksDiskAttachments) (attachments []map[string]interface{}) {
	if attach == nil {
		return attachments
	}
	for _, att := range attach.Attachment {
		attachments = append(attachments, map[string]interface{} {
			"InstanceId": utils.SafeString(att.InstanceId),
			"AttachedTime": utils.SafeString(att.AttachedTime),
			"Device": utils.SafeString(att.AttachedTime),
		})
	}
	return attachments
}
