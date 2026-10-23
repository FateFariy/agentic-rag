package api

import (
	"fmt"

	"github.com/swiftbit/know-agent/pkg/utils"
)

func (r *KnowledgeScopeSaveReq) Validate() (err error) {
	if utils.IsBlank(r.ScopeName) {
		return fmt.Errorf("scope_name 不能为空")
	}
	return nil
}

func (r *KnowledgeTopicSaveReq) Validate() (err error) {
	if utils.IsBlank(r.TopicName) {
		return fmt.Errorf("topic_name 不能为空")
	}
	return nil
}
