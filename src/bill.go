package src

import (
	"errors"
	"fmt"
	bssopenapi20171214 "github.com/alibabacloud-go/bssopenapi-20171214/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"time"
)

var InstanceBillSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "billing_cycle",
		Required: true,
		Type: utils.String,
	},
	utils.Scheme{
		Param: "is_hide_zero_charge",
		Required: false,
		Type: utils.Bool,
		Default: false,
	},
	utils.Scheme{
		Param: "subscription_type",
		Required: false,
		Type: utils.String,
		Default: "",
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
		Default: "",
	},
}

type InstanceBill struct {
	client *bssopenapi20171214.Client
	credential input.Credential
	input.Resource
}

func (bill *InstanceBill)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("business.aliyuncs.com")
	cli, err := bssopenapi20171214.NewClient(config)
	if err != nil {
		return err
	}
	bill.credential = credential
	bill.client = cli
	return nil
}

func (InstanceBill)Call(params input.Params, replay *input.Replay) error {
	bill := &InstanceBill{}
	var next string
	var err error
	var bills []interface{}
	params.Args, err = utils.CheckParam(params.Args, InstanceBillSchemes)
	if err != nil {
		return err
	}
	err = bill.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	billingCycle := params.Args["billing_cycle"].(string)
	if billingCycle == "current" {
		now := time.Now()
		billingCycle = now.Format("2006") + "-" + now.Format("01")
	}
	hideZeroCharge := params.Args["is_hide_zero_charge"].(bool)
	subscription := params.Args["subscription_type"].(string)
	query := map[string]interface{} {
		"BillingCycle": billingCycle,
		"CloudType": common.CloudType,
		"AccountId": bill.credential.AccountId,
	}
	request := &bssopenapi20171214.DescribeInstanceBillRequest{
		BillingCycle: &billingCycle,
		MaxResults: &limit,
		NextToken: &next,
		IsHideZeroCharge: &hideZeroCharge,
	}
	if subscription != "" {
		request.SubscriptionType = &subscription
	}
	resp, err := bill.client.DescribeInstanceBill(request)
	if err != nil {
		return err
	}
	if resp.Body.Success == nil || (resp.Body.Success != nil && !(*resp.Body.Success)) {
		return errors.New(utils.SafeString(resp.Body.Message))
	}
	if resp.Body.Data == nil || resp.Body.Data.Items == nil {
		return errors.New("bad response for query instance bill")
	}
	for _, b := range resp.Body.Data.Items {
		iBill := models.NewInstanceBillModel()
		iBill.Deleted = 0
		iBill.CloudType = common.CloudType
		iBill.AccountId = bill.credential.AccountId
		iBill.Region = utils.SafeString(b.Region)
		iBill.InstanceId = utils.SafeString(b.InstanceID)
		iBill.InstanceName = utils.SafeString(b.NickName)
		iBill.BillingCycle = billingCycle
		iBill.BillingDate = utils.SafeString(b.BillingDate)
		iBill.SubscriptionType = utils.SafeString(b.SubscriptionType)
		iBill.ProductCode = utils.SafeString(b.ProductCode)
		iBill.ProductName = utils.SafeString(b.ProductName)
		iBill.ItemAction = utils.SafeString(b.Item)
		iBill.PretaxGrossAmount = float64(utils.SafeFloat32(b.PretaxGrossAmount))
		iBill.PretaxAmount = float64(utils.SafeFloat32(b.PretaxAmount))
		iBill.DeductionAmount = iBill.PretaxGrossAmount - iBill.PretaxAmount
		iBill.Tags = utils.SafeString(b.Tag)
		iBill.Extra = map[string]interface{}{
			"InstanceConfig": utils.SafeString(b.InstanceConfig),
			"InstanceSpec": utils.SafeString(b.InstanceSpec),
			"DeductedByCashCoupons": utils.SafeFloat32(b.DeductedByCashCoupons),
			"DeductedByPrepaidCard": utils.SafeFloat32(b.DeductedByPrepaidCard),
			"DeductedByCoupons": utils.SafeFloat32(b.DeductedByCoupons),
			"DeductedByResourcePackage": utils.SafeString(b.DeductedByResourcePackage),
			"OutstandingAmount": utils.SafeFloat32(b.OutstandingAmount),
			"ResourceGroup": utils.SafeString(b.ResourceGroup),
			"Zone": utils.SafeString(b.Zone),
			"Currency": utils.SafeString(b.Currency),
		}
		iBill.SetIndex()
		iBill.SetChecksum()
		checked, key := iBill.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		bills = append(bills, iBill)
	}
	if !utils.CheckQueryKeys(query, models.InstanceBillModel{}) {
		return errors.New("query key is not attribute of InstanceBillModel")
	}
	next = utils.SafeString(resp.Body.Data.NextToken)
	replay.Next = next
	replay.Query = query
	replay.Result = bills
	return nil
}
