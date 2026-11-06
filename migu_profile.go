package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var miguProfileURL = "https://app.u.nf.migu.cn/pc/user/h5/queryUserInfo/v1.0"

type miguProfile struct {
	UserID    string
	Nickname  string
	AvatarURL string
	Phone     string
	LevelName string
	MedalName string
}

type miguProfileResponse struct {
	Code         string          `json:"code"`
	Info         string          `json:"info"`
	UserInfoItem json.RawMessage `json:"userInfoItem"`
	Data         json.RawMessage `json:"data"`
}

type miguProfileItem struct {
	UserID        string `json:"userId"`
	NickName      string `json:"nickName"`
	BigIcon       string `json:"bigIcon"`
	MiddleIcon    string `json:"middleIcon"`
	SmallIcon     string `json:"smallIcon"`
	Msisdn        string `json:"msisdn"`
	UserType      string `json:"userType"`
	UserLevelInfo struct {
		Level     string `json:"level"`
		LevelName string `json:"levelName"`
	} `json:"userLevelInfo"`
	UserMedalInfoList []struct {
		MedalName string `json:"medalName"`
	} `json:"userMedalInfoList"`
}

func fetchMiguProfile(ctx context.Context, token string) (miguProfile, error) {
	token = normalizeMiguToken(token)
	if token == "" {
		return miguProfile{}, errMiguTokenStale
	}
	body, err := miguGet(ctx, miguProfileURL, token)
	if err != nil {
		return miguProfile{}, err
	}

	var response miguProfileResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return miguProfile{}, fmt.Errorf("解析咪咕账号资料失败：%w", err)
	}
	if response.Code != "000000" {
		if info := strings.TrimSpace(response.Info); info != "" {
			return miguProfile{}, errors.New(info)
		}
		if response.Code == "" {
			return miguProfile{}, errors.New("咪咕账号资料接口没有返回结果")
		}
		return miguProfile{}, fmt.Errorf("咪咕账号资料接口返回 %q", response.Code)
	}

	item := decodeMiguProfileItem(response.UserInfoItem)
	if !item.hasData() {
		item = decodeMiguProfileItem(response.Data)
	}
	if !item.hasData() {
		return miguProfile{}, errors.New("咪咕没有返回账号资料")
	}
	return item.profile(), nil
}

func decodeMiguProfileItem(raw json.RawMessage) miguProfileItem {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return miguProfileItem{}
	}
	var item miguProfileItem
	if json.Unmarshal(raw, &item) == nil && item.hasData() {
		return item
	}
	var wrapper struct {
		UserInfoItem json.RawMessage `json:"userInfoItem"`
	}
	if json.Unmarshal(raw, &wrapper) == nil && len(wrapper.UserInfoItem) > 0 {
		_ = json.Unmarshal(wrapper.UserInfoItem, &item)
	}
	return item
}

func (item miguProfileItem) hasData() bool {
	return strings.TrimSpace(item.UserID) != "" ||
		strings.TrimSpace(item.NickName) != "" ||
		strings.TrimSpace(item.Msisdn) != ""
}

func (item miguProfileItem) profile() miguProfile {
	profile := miguProfile{
		UserID:    strings.TrimSpace(item.UserID),
		Nickname:  strings.TrimSpace(item.NickName),
		AvatarURL: miguImage(firstNonEmpty(item.BigIcon, item.MiddleIcon, item.SmallIcon)),
		Phone:     maskMiguPhone(item.Msisdn),
		LevelName: strings.TrimSpace(firstNonEmpty(item.UserLevelInfo.LevelName, item.UserLevelInfo.Level)),
	}
	if len(item.UserMedalInfoList) > 0 {
		profile.MedalName = strings.TrimSpace(item.UserMedalInfoList[0].MedalName)
	}
	return profile
}

func maskMiguPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	var digits []byte
	for i := 0; i < len(phone); i++ {
		if phone[i] >= '0' && phone[i] <= '9' {
			digits = append(digits, phone[i])
		}
	}
	if len(digits) == 11 {
		return string(digits[:3]) + "****" + string(digits[7:])
	}
	return phone
}

func miguProfileInitial(name string) string {
	for _, r := range strings.TrimSpace(name) {
		return string(r)
	}
	return "咪"
}

func (a *app) loadMiguProfile(token string) {
	token = normalizeMiguToken(token)
	if token == "" {
		a.miguProfile = nil
		a.miguProfileErr = ""
		a.miguProfileLoading = false
		return
	}
	a.miguProfileLoading = true
	a.miguProfileErr = ""
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		profile, err := fetchMiguProfile(ctx, token)
		a.post(func() {
			if normalizeMiguToken(a.miguToken) != token {
				return
			}
			a.miguProfileLoading = false
			if err != nil {
				a.miguProfileErr = friendlyError(err)
				a.refresh()
				return
			}
			a.miguProfile = &profile
			a.miguProfileErr = ""
			if profile.AvatarURL != "" {
				a.requestArt(profile.AvatarURL)
			}
			a.refresh()
		})
	}()
}
