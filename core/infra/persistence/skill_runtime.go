package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"aivo/core/domain"
)

func (s *Store) SaveSkill(ctx context.Context, skill domain.SkillEntry) (domain.SkillEntry, error) {
	now := skill.TimeUpdated
	if now == "" {
		now = domain.NowString(time.Now())
	}
	if strings.TrimSpace(skill.ID) == "" {
		skill.ID = uuid.NewString()
	}
	if strings.TrimSpace(skill.TimeCreated) == "" {
		skill.TimeCreated = now
	}
	skill.TimeUpdated = now
	row := skillRow{
		ID: skill.ID, Name: strings.TrimSpace(skill.Name), Description: strings.TrimSpace(skill.Description),
		Scope: strings.TrimSpace(skill.Scope), Source: strings.TrimSpace(skill.Source), RootPath: strings.TrimSpace(skill.RootPath),
		SkillPath: strings.TrimSpace(skill.SkillPath), ContentHash: strings.TrimSpace(skill.ContentHash),
		Enabled: boolInt(skill.Enabled), Metadata: encodeStringMap(skill.Metadata), TimeCreated: skill.TimeCreated, TimeUpdated: skill.TimeUpdated,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.SkillEntry{}, err
	}
	return skillFromRow(row), nil
}

func (s *Store) GetSkill(ctx context.Context, id string) (domain.SkillEntry, error) {
	var row skillRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.SkillEntry{}, err
	}
	return skillFromRow(row), nil
}

func (s *Store) GetSkillByName(ctx context.Context, name string, scope string) (domain.SkillEntry, error) {
	q := s.db.WithContext(ctx).Model(&skillRow{}).Where("name = ?", strings.TrimSpace(name))
	if strings.TrimSpace(scope) != "" {
		q = q.Where("scope = ?", strings.TrimSpace(scope))
	}
	var row skillRow
	if err := q.Order("CASE scope WHEN 'project' THEN 0 WHEN 'global' THEN 1 ELSE 2 END").First(&row).Error; err != nil {
		return domain.SkillEntry{}, err
	}
	return skillFromRow(row), nil
}

func (s *Store) ListSkills(ctx context.Context, includeDisabled bool) ([]domain.SkillEntry, error) {
	q := s.db.WithContext(ctx).Model(&skillRow{})
	if !includeDisabled {
		q = q.Where("enabled = ?", 1)
	}
	var rows []skillRow
	if err := q.Order("scope ASC, name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SkillEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, skillFromRow(row))
	}
	return out, nil
}

func (s *Store) SetSkillEnabled(ctx context.Context, id string, enabled bool) (domain.SkillEntry, error) {
	now := domain.NowString(time.Now())
	if err := s.db.WithContext(ctx).Model(&skillRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"enabled": boolInt(enabled), "time_updated": now,
	}).Error; err != nil {
		return domain.SkillEntry{}, err
	}
	return s.GetSkill(ctx, id)
}

func (s *Store) DeleteSkill(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("skill_id = ?", id).Delete(&skillSourceRow{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&skillRow{}).Error
	})
}

func (s *Store) SaveSkillSource(ctx context.Context, source domain.SkillSource) (domain.SkillSource, error) {
	if strings.TrimSpace(source.ID) == "" {
		source.ID = uuid.NewString()
	}
	row := skillSourceRow{
		ID: source.ID, SkillID: source.SkillID, Source: source.Source, Scope: source.Scope, RootPath: source.RootPath,
		SkillPath: source.SkillPath, ContentHash: source.ContentHash, LastSeenAt: source.LastSeenAt,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.SkillSource{}, err
	}
	return skillSourceFromRow(row), nil
}

func (s *Store) ListSkillSources(ctx context.Context, skillID string) ([]domain.SkillSource, error) {
	var rows []skillSourceRow
	if err := s.db.WithContext(ctx).Where("skill_id = ?", strings.TrimSpace(skillID)).Order("last_seen_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SkillSource, 0, len(rows))
	for _, row := range rows {
		out = append(out, skillSourceFromRow(row))
	}
	return out, nil
}

func (s *Store) SaveSkillImportCandidate(ctx context.Context, candidate domain.SkillImportCandidate) (domain.SkillImportCandidate, error) {
	if strings.TrimSpace(candidate.ID) == "" {
		candidate.ID = uuid.NewString()
	}
	row := skillImportCandidateRow{
		ID: candidate.ID, Name: candidate.Name, Description: candidate.Description, Scope: candidate.Scope, Source: candidate.Source,
		RootPath: candidate.RootPath, SkillPath: candidate.SkillPath, ContentHash: candidate.ContentHash,
		Status: candidate.Status, ConflictID: candidate.ConflictID, Error: candidate.Error, LastSeenAt: candidate.LastSeenAt,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "root_path"}, {Name: "skill_path"}}, UpdateAll: true}).Create(&row).Error; err != nil {
		return domain.SkillImportCandidate{}, err
	}
	return skillCandidateFromRow(row), nil
}

func (s *Store) GetSkillImportCandidate(ctx context.Context, id string) (domain.SkillImportCandidate, error) {
	var row skillImportCandidateRow
	if err := s.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&row).Error; err != nil {
		return domain.SkillImportCandidate{}, err
	}
	return skillCandidateFromRow(row), nil
}

func (s *Store) ListSkillImportCandidates(ctx context.Context, includeIgnored bool) ([]domain.SkillImportCandidate, error) {
	q := s.db.WithContext(ctx).Model(&skillImportCandidateRow{})
	if !includeIgnored {
		q = q.Where("status <> ?", domain.SkillCandidateStatusIgnored)
	}
	var rows []skillImportCandidateRow
	if err := q.Order("last_seen_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SkillImportCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, skillCandidateFromRow(row))
	}
	return out, nil
}

func (s *Store) MarkSkillImportCandidateStatus(ctx context.Context, id string, status string, conflictID string, errText string) (domain.SkillImportCandidate, error) {
	if strings.TrimSpace(id) == "" {
		return domain.SkillImportCandidate{}, errors.New("candidate id is required")
	}
	if err := s.db.WithContext(ctx).Model(&skillImportCandidateRow{}).Where("id = ?", strings.TrimSpace(id)).Updates(map[string]any{
		"status": status, "conflict_id": conflictID, "error": errText,
	}).Error; err != nil {
		return domain.SkillImportCandidate{}, err
	}
	return s.GetSkillImportCandidate(ctx, id)
}

func (s *Store) MarkSkillImportCandidatesByNameStatus(ctx context.Context, name string, status string, errText string) ([]domain.SkillImportCandidate, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("skill name is required")
	}
	if err := s.db.WithContext(ctx).Model(&skillImportCandidateRow{}).Where("name = ?", name).Updates(map[string]any{
		"status": status, "conflict_id": "", "error": errText,
	}).Error; err != nil {
		return nil, err
	}
	var rows []skillImportCandidateRow
	if err := s.db.WithContext(ctx).Where("name = ?", name).Order("last_seen_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.SkillImportCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, skillCandidateFromRow(row))
	}
	return out, nil
}

func skillFromRow(row skillRow) domain.SkillEntry {
	return domain.SkillEntry{
		ID: row.ID, Name: row.Name, Description: row.Description, Scope: row.Scope, Source: row.Source,
		RootPath: row.RootPath, SkillPath: row.SkillPath, ContentHash: row.ContentHash, Enabled: row.Enabled != 0,
		Metadata: decodeStringMap(row.Metadata), TimeCreated: row.TimeCreated, TimeUpdated: row.TimeUpdated,
	}
}

func skillSourceFromRow(row skillSourceRow) domain.SkillSource {
	return domain.SkillSource{
		ID: row.ID, SkillID: row.SkillID, Source: row.Source, Scope: row.Scope, RootPath: row.RootPath,
		SkillPath: row.SkillPath, ContentHash: row.ContentHash, LastSeenAt: row.LastSeenAt,
	}
}

func skillCandidateFromRow(row skillImportCandidateRow) domain.SkillImportCandidate {
	return domain.SkillImportCandidate{
		ID: row.ID, Name: row.Name, Description: row.Description, Scope: row.Scope, Source: row.Source,
		RootPath: row.RootPath, SkillPath: row.SkillPath, ContentHash: row.ContentHash, Status: row.Status,
		ConflictID: row.ConflictID, Error: row.Error, LastSeenAt: row.LastSeenAt,
	}
}
