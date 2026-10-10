package model

import (
	"errors"

	"github.com/axzcnzxis/myapi/common"

	"gorm.io/gorm"
)

// Announcement lifecycle statuses.
const (
	AnnouncementStatusDraft     = "draft"
	AnnouncementStatusScheduled = "scheduled"
	AnnouncementStatusPublished = "published"
	AnnouncementStatusCancelled = "cancelled"
	AnnouncementStatusExpired   = "expired"
)

// Announcement delivery modes (公告的方式).
const (
	AnnouncementModeNotice = "notice" // bell / timeline only
	AnnouncementModePopup  = "popup"  // modal popup only
	AnnouncementModeBoth   = "both"   // both notice and popup
)

// Announcement is a managed platform announcement. Times are stored as UTC
// Unix seconds following the project-wide convention; the console renders them
// in Beijing time (UTC+8).
type Announcement struct {
	Id            int64  `json:"id" gorm:"primary_key"`
	Title         string `json:"title" gorm:"type:varchar(200);index"`
	Content       string `json:"content" gorm:"type:text"`
	Type          string `json:"type" gorm:"type:varchar(20);default:'default'"`
	Mode          string `json:"mode" gorm:"type:varchar(20);default:'notice';index"`
	Status        string `json:"status" gorm:"type:varchar(20);index;default:'draft'"`
	PublishAt     int64  `json:"publish_at" gorm:"bigint;index"`
	PublishedAt   int64  `json:"published_at" gorm:"bigint;index"`
	ExpireAt      int64  `json:"expire_at" gorm:"bigint;index"`
	DailyLimit    int    `json:"daily_limit" gorm:"default:0"`
	ShowStartTime string `json:"show_start_time" gorm:"type:varchar(5)"`
	ShowEndTime   string `json:"show_end_time" gorm:"type:varchar(5)"`
	PublisherId   int    `json:"publisher_id" gorm:"index"`
	PublisherName string `json:"publisher_name" gorm:"type:varchar(64)"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint;index"`
}

func (a *Announcement) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if a.CreatedAt == 0 {
		a.CreatedAt = now
	}
	if a.UpdatedAt == 0 {
		a.UpdatedAt = now
	}
	return nil
}

// AnnouncementFilter narrows the admin listing.
type AnnouncementFilter struct {
	Scope  string `form:"scope"`  // current | history | all (default all)
	Status string `form:"status"` // exact status filter
}

func (filter AnnouncementFilter) query() *gorm.DB {
	query := DB.Model(&Announcement{})
	switch filter.Scope {
	case "current":
		query = query.Where("status = ?", AnnouncementStatusPublished)
	case "history":
		query = query.Where("status IN ?", []string{
			AnnouncementStatusCancelled,
			AnnouncementStatusExpired,
			AnnouncementStatusDraft,
		})
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	return query
}

func CreateAnnouncement(a *Announcement) error {
	return DB.Create(a).Error
}

// UpdateAnnouncement persists the full record and refreshes updated_at.
func UpdateAnnouncement(a *Announcement) error {
	a.UpdatedAt = common.GetTimestamp()
	return DB.Save(a).Error
}

func DeleteAnnouncementById(id int64) error {
	return DB.Delete(&Announcement{}, "id = ?", id).Error
}

func GetAnnouncementById(id int64) (*Announcement, error) {
	var a Announcement
	if err := DB.Where("id = ?", id).First(&a).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func ListAnnouncements(filter AnnouncementFilter, offset, limit int) ([]*Announcement, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	query := filter.query()
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*Announcement
	err := query.Order("id desc").Offset(max(0, offset)).Limit(min(limit, 100)).Find(&list).Error
	return list, total, err
}

// GetCurrentAnnouncements returns every published, non-expired announcement,
// newest first, for the bell/timeline and the dashboard card.
func GetCurrentAnnouncements(now int64) ([]*Announcement, error) {
	var list []*Announcement
	err := DB.Where("status = ? AND (expire_at = 0 OR expire_at > ?)", AnnouncementStatusPublished, now).
		Order("published_at desc, id desc").
		Find(&list).Error
	return list, err
}

// GetActivePopupAnnouncements returns every published, non-expired popup
// announcement, newest first. The caller applies the per-day show window so an
// out-of-window newest item does not mask an older in-window one.
func GetActivePopupAnnouncements(now int64) ([]*Announcement, error) {
	var list []*Announcement
	err := DB.Where("status = ? AND mode IN ? AND (expire_at = 0 OR expire_at > ?)",
		AnnouncementStatusPublished, []string{AnnouncementModePopup, AnnouncementModeBoth}, now).
		Order("published_at desc, id desc").
		Find(&list).Error
	return list, err
}

// PublishDueAnnouncements flips scheduled announcements whose publish time has
// arrived to published. It is driven by the periodic maintenance task.
func PublishDueAnnouncements(now int64) (int64, error) {
	result := DB.Model(&Announcement{}).
		Where("status = ? AND publish_at > 0 AND publish_at <= ?", AnnouncementStatusScheduled, now).
		Updates(map[string]any{
			"status":       AnnouncementStatusPublished,
			"published_at": now,
			"updated_at":   now,
		})
	return result.RowsAffected, result.Error
}

// ExpireDueAnnouncements flips published announcements past their expire time
// to expired so they drop out of the current list and popups.
func ExpireDueAnnouncements(now int64) (int64, error) {
	result := DB.Model(&Announcement{}).
		Where("status = ? AND expire_at > 0 AND expire_at <= ?", AnnouncementStatusPublished, now).
		Updates(map[string]any{
			"status":     AnnouncementStatusExpired,
			"updated_at": now,
		})
	return result.RowsAffected, result.Error
}
