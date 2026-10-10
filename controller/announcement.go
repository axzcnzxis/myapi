package controller

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/axzcnzxis/myapi/common"
	"github.com/axzcnzxis/myapi/model"
	"github.com/axzcnzxis/myapi/setting/console_setting"

	"github.com/gin-gonic/gin"
)

var (
	announcementValidTypes = map[string]bool{
		"default": true, "ongoing": true, "success": true, "warning": true, "error": true,
	}
	announcementValidModes = map[string]bool{
		model.AnnouncementModeNotice: true,
		model.AnnouncementModePopup:  true,
		model.AnnouncementModeBoth:   true,
	}
	announcementClockPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

	// beijingLocation is used to evaluate the per-day show window (展示时段),
	// which administrators configure in Beijing time.
	beijingLocation = time.FixedZone("CST", 8*3600)
)

// AnnouncementRequest is the admin create/update payload.
type AnnouncementRequest struct {
	Id            int64  `json:"id"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	Type          string `json:"type"`
	Mode          string `json:"mode"`
	PublishAt     int64  `json:"publish_at"`
	ExpireAt      int64  `json:"expire_at"`
	DailyLimit    int    `json:"daily_limit"`
	ShowStartTime string `json:"show_start_time"`
	ShowEndTime   string `json:"show_end_time"`
}

func parseClockMinutes(value string) (int, bool) {
	if !announcementClockPattern.MatchString(value) {
		return 0, false
	}
	parts := strings.SplitN(value, ":", 2)
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	return hour*60 + minute, true
}

// withinShowWindow reports whether now falls inside the announcement's daily
// show window, interpreted in Beijing time. An empty or degenerate window means
// "all day".
func withinShowWindow(a *model.Announcement, now int64) bool {
	if a.ShowStartTime == "" || a.ShowEndTime == "" {
		return true
	}
	start, okStart := parseClockMinutes(a.ShowStartTime)
	end, okEnd := parseClockMinutes(a.ShowEndTime)
	if !okStart || !okEnd || start == end {
		return true
	}
	beijing := time.Unix(now, 0).In(beijingLocation)
	minutes := beijing.Hour()*60 + beijing.Minute()
	if start < end {
		return minutes >= start && minutes < end
	}
	// Window crosses midnight, e.g. 22:00 - 02:00.
	return minutes >= start || minutes < end
}

// validateAnnouncementRequest normalizes and validates the payload, returning a
// user-facing error message when invalid.
func validateAnnouncementRequest(req *AnnouncementRequest) (string, bool) {
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		return "公告标题不能为空", false
	}
	if utf8.RuneCountInString(req.Title) > 200 {
		return "公告标题长度不能超过200个字符", false
	}
	if req.Content == "" {
		return "公告内容不能为空", false
	}
	if utf8.RuneCountInString(req.Content) > 5000 {
		return "公告内容长度不能超过5000个字符", false
	}
	if req.Type == "" {
		req.Type = "default"
	}
	if !announcementValidTypes[req.Type] {
		return "公告类型不合法", false
	}
	if req.Mode == "" {
		req.Mode = model.AnnouncementModeNotice
	}
	if !announcementValidModes[req.Mode] {
		return "公告方式不合法", false
	}
	if req.DailyLimit < 0 {
		return "每日展示次数不能为负数", false
	}
	req.ShowStartTime = strings.TrimSpace(req.ShowStartTime)
	req.ShowEndTime = strings.TrimSpace(req.ShowEndTime)
	if (req.ShowStartTime == "") != (req.ShowEndTime == "") {
		return "展示时段的开始与结束时间必须同时填写", false
	}
	if req.ShowStartTime != "" {
		if _, ok := parseClockMinutes(req.ShowStartTime); !ok {
			return "展示时段开始时间格式错误，应为 HH:MM", false
		}
		if _, ok := parseClockMinutes(req.ShowEndTime); !ok {
			return "展示时段结束时间格式错误，应为 HH:MM", false
		}
	}
	now := common.GetTimestamp()
	if req.PublishAt < 0 {
		return "发布时间不合法", false
	}
	if req.ExpireAt != 0 {
		effectivePublish := req.PublishAt
		if effectivePublish == 0 {
			effectivePublish = now
		}
		if req.ExpireAt <= effectivePublish {
			return "过期时间必须晚于发布时间", false
		}
	}
	return "", true
}

// GetAnnouncements lists announcements for the admin console, supporting the
// current / history scopes and an exact status filter.
func GetAnnouncements(c *gin.Context) {
	filter := model.AnnouncementFilter{
		Scope:  c.Query("scope"),
		Status: c.Query("status"),
	}
	pageInfo := common.GetPageQuery(c)
	list, total, err := model.ListAnnouncements(filter, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(list)
	common.ApiSuccess(c, pageInfo)
}

func GetAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "公告 ID 不合法")
		return
	}
	announcement, err := model.GetAnnouncementById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if announcement == nil {
		common.ApiErrorMsg(c, "公告不存在")
		return
	}
	common.ApiSuccess(c, announcement)
}

func AddAnnouncement(c *gin.Context) {
	req := AnnouncementRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if msg, ok := validateAnnouncementRequest(&req); !ok {
		common.ApiErrorMsg(c, msg)
		return
	}
	now := common.GetTimestamp()
	announcement := model.Announcement{
		Title:         req.Title,
		Content:       req.Content,
		Type:          req.Type,
		Mode:          req.Mode,
		PublishAt:     req.PublishAt,
		ExpireAt:      req.ExpireAt,
		DailyLimit:    req.DailyLimit,
		ShowStartTime: req.ShowStartTime,
		ShowEndTime:   req.ShowEndTime,
		PublisherId:   c.GetInt("id"),
		PublisherName: c.GetString("username"),
	}
	// An empty or past publish time means "publish right away".
	if req.PublishAt == 0 || req.PublishAt <= now {
		announcement.Status = model.AnnouncementStatusPublished
		announcement.PublishedAt = now
	} else {
		announcement.Status = model.AnnouncementStatusScheduled
	}
	if err := model.CreateAnnouncement(&announcement); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "announcement.create", map[string]any{
		"id":     announcement.Id,
		"title":  announcement.Title,
		"mode":   announcement.Mode,
		"status": announcement.Status,
	})
	common.ApiSuccess(c, announcement)
}

func UpdateAnnouncement(c *gin.Context) {
	req := AnnouncementRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Id == 0 {
		common.ApiErrorMsg(c, "公告 ID 不能为空")
		return
	}
	announcement, err := model.GetAnnouncementById(req.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if announcement == nil {
		common.ApiErrorMsg(c, "公告不存在")
		return
	}
	if msg, ok := validateAnnouncementRequest(&req); !ok {
		common.ApiErrorMsg(c, msg)
		return
	}
	now := common.GetTimestamp()
	announcement.Title = req.Title
	announcement.Content = req.Content
	announcement.Type = req.Type
	announcement.Mode = req.Mode
	announcement.PublishAt = req.PublishAt
	announcement.ExpireAt = req.ExpireAt
	announcement.DailyLimit = req.DailyLimit
	announcement.ShowStartTime = req.ShowStartTime
	announcement.ShowEndTime = req.ShowEndTime
	// Recompute lifecycle for not-yet-published announcements. Already published
	// or cancelled announcements keep their state and only refresh content.
	if announcement.Status == model.AnnouncementStatusDraft || announcement.Status == model.AnnouncementStatusScheduled {
		if req.PublishAt == 0 || req.PublishAt <= now {
			announcement.Status = model.AnnouncementStatusPublished
			announcement.PublishedAt = now
		} else {
			announcement.Status = model.AnnouncementStatusScheduled
			announcement.PublishedAt = 0
		}
	}
	if err := model.UpdateAnnouncement(announcement); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "announcement.update", map[string]any{
		"id":     announcement.Id,
		"title":  announcement.Title,
		"status": announcement.Status,
	})
	common.ApiSuccess(c, announcement)
}

func DeleteAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "公告 ID 不合法")
		return
	}
	if err := model.DeleteAnnouncementById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "announcement.delete", map[string]any{"id": id})
	common.ApiSuccess(c, nil)
}

// PublishAnnouncement publishes a draft/scheduled announcement immediately.
func PublishAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "公告 ID 不合法")
		return
	}
	announcement, err := model.GetAnnouncementById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if announcement == nil {
		common.ApiErrorMsg(c, "公告不存在")
		return
	}
	if announcement.Status == model.AnnouncementStatusCancelled {
		common.ApiErrorMsg(c, "已取消的公告不能再次发布")
		return
	}
	now := common.GetTimestamp()
	announcement.Status = model.AnnouncementStatusPublished
	if announcement.PublishedAt == 0 {
		announcement.PublishedAt = now
	}
	if err := model.UpdateAnnouncement(announcement); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "announcement.publish", map[string]any{"id": announcement.Id})
	common.ApiSuccess(c, announcement)
}

// CancelAnnouncement cancels the currently published or scheduled announcement.
func CancelAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "公告 ID 不合法")
		return
	}
	announcement, err := model.GetAnnouncementById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if announcement == nil {
		common.ApiErrorMsg(c, "公告不存在")
		return
	}
	if announcement.Status != model.AnnouncementStatusPublished && announcement.Status != model.AnnouncementStatusScheduled {
		common.ApiErrorMsg(c, "只有已发布或待发布的公告可以取消")
		return
	}
	announcement.Status = model.AnnouncementStatusCancelled
	if err := model.UpdateAnnouncement(announcement); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "announcement.cancel", map[string]any{"id": announcement.Id})
	common.ApiSuccess(c, announcement)
}

// GetPublicAnnouncements serves the current published announcements to the
// console (bell timeline / dashboard card). Public so the home page can render
// them before login.
func GetPublicAnnouncements(c *gin.Context) {
	now := common.GetTimestamp()
	list, err := model.GetCurrentAnnouncements(now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

// buildStatusAnnouncements returns the announcements surfaced through
// /api/status for the bell timeline and dashboard card. Published notice/both
// announcements from the new table take precedence; the legacy console_setting
// list is used only while the new feature holds no data, so existing
// configurations keep working.
func buildStatusAnnouncements() []map[string]interface{} {
	now := common.GetTimestamp()
	list, err := model.GetCurrentAnnouncements(now)
	if err == nil {
		items := make([]map[string]interface{}, 0, len(list))
		for _, a := range list {
			if a.Mode == model.AnnouncementModePopup {
				continue
			}
			items = append(items, map[string]interface{}{
				"id":          a.Id,
				"title":       a.Title,
				"content":     a.Content,
				"type":        a.Type,
				"publishDate": time.Unix(a.PublishedAt, 0).UTC().Format(time.RFC3339),
				"extra":       a.PublisherName,
			})
		}
		if len(list) > 0 {
			// The new feature is in use (even popup-only); do not fall back.
			return items
		}
	}
	return console_setting.GetAnnouncements()
}

// GetPublicPopupAnnouncement serves the newest popup announcement that is
// eligible right now (published, not expired, inside its daily show window).
// Returns data: null when nothing should be shown. Public for the pre-login
// home page.
func GetPublicPopupAnnouncement(c *gin.Context) {
	now := common.GetTimestamp()
	list, err := model.GetActivePopupAnnouncements(now)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, a := range list {
		if withinShowWindow(a, now) {
			common.ApiSuccess(c, a)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    nil,
	})
}
