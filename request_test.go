package openai

import (
	"encoding/json"
	"fmt"
	"testing"
)

type MyArrContentItem struct {
	Image string `json:"image"`
}

func (m *MyArrContentItem) GetType() string {
	return "image"
}

func (m *MyArrContentItem) Keyword() string {
	return "image"
}

func (m *MyArrContentItem) CastToUserArrTextContent() (UserTextContent, bool) {
	return UserTextContent{}, false
}

func (m *MyArrContentItem) CastToUserArrImgContent() (UserImgContent, bool) {
	return UserImgContent{
		Type: "image_url",
		ImageURL: ImgURL{
			URL: m.Image,
		},
	}, true
}

// GetText implements UserArrContentItem.
func (m *MyArrContentItem) GetText() string {
	return m.Image
}

type MyArrContentItemMatcher struct{}

func (m MyArrContentItemMatcher) MatchContentItem(keyword string) UserArrContentItem {
	defaultMatcher := DefaultArrMsgContentItemMatcher{}
	if item := defaultMatcher.MatchContentItem(keyword); item != nil {
		return item
	}
	if keyword == "image" {
		return &MyArrContentItem{}
	}
	return nil
}

func TestMessage_UnmarshalJSON(t *testing.T) {
	choice := new(ChatChoice)
	respByts := []byte(`{
	"index": 0,
	"message": {
		"role": "user",
		"content": "你好",
		"name": "test",
		"refusal": "拒绝"
	},
	"finish_reason": "stop"
}`)

	err := json.Unmarshal(respByts, choice)
	if err != nil {
		t.Error(err)
		return
	}
	choiceByts, err := json.Marshal(choice)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(choiceByts))

	SetArrMsgContentItemMatcher(MyArrContentItemMatcher{})
	respByts = []byte(`{
		"index": 0,
		"message": {
			"role": "user",
			"content": [
				{"type": "text", "text": "What’s in this image?"},
				{
					"type": "image_url",
					"image_url": {
						"url": "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg",
						"detail": "high"
					}
				},
				{"image": "https://dashscope.oss-cn-beijing.aliyuncs.com/images/dog_and_girl.jpeg"},
				{"text": "这个图片是哪里？"}
			],
			"name": "test",
			"refusal": "拒绝"
		},
		"finish_reason": "stop"
	}`)

	err = json.Unmarshal(respByts, choice)
	if err != nil {
		t.Error(err)
		return
	}
	choiceByts, err = json.Marshal(choice)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(string(choiceByts))
}
