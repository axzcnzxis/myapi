package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/logger"
	"github.com/axzcnzxis/myapi/model"

	"github.com/bytedance/gopkg/util/gopool"
)

// announcementTaskTickInterval controls how quickly a scheduled announcement is
// published and how promptly an expired one drops out of the current list.
const announcementTaskTickInterval = 30 * time.Second

var announcementTaskOnce sync.Once

// StartAnnouncementTask runs the announcement maintenance loop on the master
// node: it publishes scheduled announcements whose time has arrived and expires
// published announcements past their expire time.
func StartAnnouncementTask() {
	announcementTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("announcement task started: tick=%s", announcementTaskTickInterval))
			ticker := time.NewTicker(announcementTaskTickInterval)
			defer ticker.Stop()

			runAnnouncementOnce()
			for range ticker.C {
				runAnnouncementOnce()
			}
		})
	})
}

func runAnnouncementOnce() {
	ctx := context.Background()
	now := common.GetTimestamp()
	if n, err := model.PublishDueAnnouncements(now); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("announcement publish task failed: %v", err))
	} else if n > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("announcement publish task: published %d announcement(s)", n))
	}
	if n, err := model.ExpireDueAnnouncements(now); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("announcement expire task failed: %v", err))
	} else if n > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("announcement expire task: expired %d announcement(s)", n))
	}
}
