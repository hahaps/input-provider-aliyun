package compute

import (
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models/compute"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"strconv"
)

var ImageSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "image_owner",
		Required: true,
		Type: utils.String,
		Default: "self",
	},
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

type Image struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (image *Image)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	image.credential = credential
	image.client = cli
	return nil
}

func (Image)Call(params input.Params, replay *input.Replay) error {
	image := &Image{}
	var next string
	var err error
	var images []interface{}
	params.Args, err = utils.CheckParam(params.Args, ImageSchemes)
	if err != nil {
		return err
	}
	err = image.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	imageOwner := params.Args["image_owner"].(string)
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
		"AccountId": image.credential.AccountId,
	}
	request := &ecs20140526.DescribeImagesRequest{
		RegionId: &region,
		ImageOwnerAlias: &imageOwner,
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	resp, err := image.client.DescribeImages(request)
	if err != nil {
		return err
	}
	if resp.Body.Images == nil || resp.Body.Images.Image == nil {
		return errors.New("bad response for query images")
	}
	for _, b := range resp.Body.Images.Image {
		img := compute.NewImageModel()
		img.Deleted = 0
		img.CloudType = common.CloudType
		img.AccountId = image.credential.AccountId
		img.RegionId = region
		img.Name = utils.SafeString(b.ImageName)
		img.ProviderId = utils.SafeString(b.ImageId)
		img.OSType = utils.SafeString(b.OSType)
		img.Status = utils.SafeString(b.Status)
		img.Tags = getImageTags(b.Tags)
		img.Size = utils.SafeInt32(b.Size)
		img.Description = utils.SafeString(b.Description)
		img.CreateTime = utils.SafeString(b.CreationTime)
		img.Extra = map[string]interface{}{
			"ImageFamily": utils.SafeString(b.ImageFamily),
			"ResourceGroupId": utils.SafeString(b.ResourceGroupId),
			"ProductCode": utils.SafeString(b.ProductCode),
			"OSName": utils.SafeString(b.OSName),
			"OSNameEn": utils.SafeString(b.OSNameEn),
			"ImageVersion": utils.SafeString(b.ImageVersion),
			"Platform": utils.SafeString(b.Platform),
		}
		img.SetIndex()
		img.SetChecksum()
		checked, key := img.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		images = append(images, img)
	}
	if !utils.CheckQueryKeys(query, compute.ImageModel{}) {
		return errors.New("query key is not attribute of ImageModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = images
	return nil
}


func getImageTags(tags *ecs20140526.DescribeImagesResponseBodyImagesImageTags) string {
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
