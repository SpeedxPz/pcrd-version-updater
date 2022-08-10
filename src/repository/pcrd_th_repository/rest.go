package pcrd_th_repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/credential"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/cryptography"
	"github.com/SpeedxPz/pcrd-version-updater/src/entity/logger"
	"github.com/SpeedxPz/pcrd-version-updater/src/use_case"
	"github.com/vmihailenco/msgpack/v5"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

type rest struct {
	client  *http.Client
	baseURL string
	salt    string
}

type restCheckGameStartParam struct {
	AppType      int64  `msgpack:"app_type" json:"app_type"`
	CampaignData string `msgpack:"campaign_data" json:"campaign_data"`
	CampaignSign string `msgpack:"campaign_sign" json:"campaign_sign"`
	CampaignUser int64  `msgpack:"campaign_user" json:"campaign_user"`
	ViewerID     string `msgpack:"viewer_id" json:"viewer_id"`
}

type restTransitionAccountData struct {
}

type restCheckGameStartResp struct {
	NowViewerID             int64                       `json:"now_viewer_id"`
	IsSetTransitionPassword bool                        `json:"is_set_transition_password"`
	NowName                 string                      `json:"now_name"`
	NowTeamLevel            int64                       `json:"now_team_level"`
	NowTutorial             bool                        `json:"now_tutorial"`
	TransitionAccountData   []restTransitionAccountData `json:"transition_account_data"`
	BundleVer               string                      `json:"bundle_ver"`
	ResourceFix             bool                        `json:"resource_fix"`
	BundleFix               bool                        `json:"bundle_fix"`
}

type restRespBody interface {
	restCheckGameStartResp
}

type restDataResp[T restRespBody] struct {
	DataHeaders restDataHeader `json:"data_headers"`
	Data        T              `json:"data"`
}

type restDataHeader struct {
	ResultCode     int64  `json:"result_code"`
	RequiredResVer string `json:"required_res_ver"`
	ShortUdid      int64  `json:"short_udid"`
	ViewerID       int64  `json:"viewer_id"`
	SID            string `json:"sid"`
	ServerTime     int64  `json:"servertime"`
}

func (r rest) GetResourceVersion(ctx context.Context, c credential.Credential, v use_case.PcrdVersion) (string, error) {
	ctx, span := tracer.Start(ctx, "pcrd_th_repository.GetResourceVersion")
	defer span.End()

	param := restCheckGameStartParam{
		AppType:      0,
		CampaignData: "",
		CampaignSign: "69fc9ddde974cc75a0756abb16b2ef35",
		CampaignUser: 157428,
		ViewerID:     fmt.Sprintf("%d", c.ViewerID),
	}

	function := "check/game_start"

	result, err := r.call(ctx, c, v, function, param)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("call error: %s", err))
		return "", err
	}

	var o restDataResp[restCheckGameStartResp]

	err = json.Unmarshal([]byte(result), &o)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("parse result error: %s", err))
		return "", fmt.Errorf("error while unmashal the response: %w", use_case.ErrDataTransform)
	}

	if len(o.DataHeaders.RequiredResVer) <= 0 {
		zap.L().Error("response not contain any version", logger.WithTraceId(ctx), zap.Any("resp", o))
		span.SetStatus(codes.Error, fmt.Sprintf("remove resource version not available: %s", use_case.ErrResVerNotAvailable))
		return "", use_case.ErrResVerNotAvailable
	}

	return o.DataHeaders.RequiredResVer, nil
}

func (r rest) call(ctx context.Context, c credential.Credential, v use_case.PcrdVersion, function string, param interface{}) (string, error) {
	ctx, span := tracer.Start(ctx, "pcrd_th_repository.call")
	defer span.End()

	endpoint := fmt.Sprintf("%s/%s?format=json", r.baseURL, function)

	headers := r.defaultHeader()
	headers["APP-VER"] = v.AppVersion
	headers["RES-VER"] = v.ResVersion
	headers["UDID"] = c.Udid
	headers["SHORT-UDID"] = fmt.Sprintf("%d", c.ShortUdid)

	hash, err := r.generateParam(c, function, param)
	if err != nil {
		zap.L().Error("generate hash param failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("generate hash param failed: %s", err))
		return "", fmt.Errorf("generate hash param failed: %w", use_case.ErrInvalidRequestParam)
	}

	headers["PARAM"] = hash

	if len(c.SessionID) > 0 {
		headers["SID"] = cryptography.MakeMD5(fmt.Sprintf("%s%s", c.SessionID, r.salt))
	} else {
		headers["SID"] = cryptography.MakeMD5(fmt.Sprintf("%s%s%s", c.ViewerID, c.Udid, r.salt))
	}

	json, err := json.Marshal(param)
	if err != nil {
		zap.L().Error("error while mashal the request", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("error while mashal the request: %s", err))
		return "", fmt.Errorf("error while mashal the request: %w", use_case.ErrDataTransform)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(json))
	if err != nil {
		zap.L().Error("create request failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("create request failed: %s", err))
		return "", fmt.Errorf("create request failed: %w", use_case.ErrRetrieveData)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := r.client.Do(req)
	if err != nil {
		zap.L().Error("request failed", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("request failed: %s", err))
		return "", fmt.Errorf("request failed: %w", use_case.ErrRetrieveData)
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		zap.L().Error("response read error", logger.WithTraceId(ctx), zap.Any("error", err))
		span.SetStatus(codes.Error, fmt.Sprintf("response read error: %s", err))
		return "", fmt.Errorf("response read error: %w", use_case.ErrRetrieveData)
	}
	defer res.Body.Close()

	return string(data), nil
}

func (r rest) generateParam(c credential.Credential, function string, param interface{}) (string, error) {
	bytes, err := msgpack.Marshal(&param)
	if err != nil {
		return "", fmt.Errorf("error while mashal the request: %w", use_case.ErrDataTransform)
	}
	sEnc := base64.StdEncoding.EncodeToString(bytes)

	pathname := fmt.Sprintf("/%s", function)
	hash := cryptography.MakeSHA1(fmt.Sprintf("%s%s%s%s", c.Udid, pathname, sEnc, c.ViewerID))
	return hash, nil
}

func (r rest) defaultHeader() map[string]string {
	return map[string]string{
		"User-Agent":           "Dalvik/2.1.0 (Linux; Android 5.1.1; SOV32 Build/32.0.D.0.282; wv)",
		"X-Unity-Version":      "2018.4.22f1",
		"Content-Type":         "application/x-www-form-urlencoded",
		"DEVICE":               "2",
		"DEVICE-ID":            "ad8a8ea1422cf6f46faa846cc2ecd220",
		"DEVICE-NAME":          "Sony E6528",
		"GRAPHICS-DEVICE-NAME": "Mali-T820",
		"PLATFORM-OS-VERSION":  "Android OS 5.1 / API-22 (29.1.A.0.101/418366884)",
		"CARRIER":              "CARRIER",
		"PLATFORM":             "2",
		"LOCALE":               "Eng",
		"BATTLE-LOGIC-VERSION": "4",
		"KEYCHAIN":             "",
		"BUNDLE-VER":           "",
	}
}

func (r rest) HealthCheck(ctx context.Context) error {
	return nil
}

func NewRest(baseURL string, salt string) use_case.PcrdTHRepository {
	c := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  true,
			MaxIdleConnsPerHost: 10,
		},
	}

	r := &rest{
		client:  c,
		baseURL: baseURL,
		salt:    salt,
	}
	return r
}
