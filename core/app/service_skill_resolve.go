package app

import (
	"context"
	"errors"
	"sort"
	"strings"
	"unicode"

	"aivo/core/domain"
)

func (s *Service) skillResolveCandidates(ctx context.Context) ([]SkillResolveCandidate, error) {
	result, err := s.ListSkills(ctx, domain.SkillListInput{})
	if err != nil {
		return nil, err
	}
	byName := map[string]SkillResolveCandidate{}
	for _, skill := range result.Entries {
		if !skill.Enabled || strings.TrimSpace(skill.Description) == "" {
			continue
		}
		byName[skill.Name] = SkillResolveCandidate{Name: skill.Name, Description: skill.Description, Scope: skill.Scope, Source: skill.Source, Status: "imported"}
	}
	out := make([]SkillResolveCandidate, 0, len(byName))
	for _, candidate := range byName {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func validateSkillResolveSelection(candidates []SkillResolveCandidate, names []string, limit int) []string {
	allowed := map[string]bool{}
	for _, candidate := range candidates {
		allowed[normalizeSkillName(candidate.Name)] = true
	}
	seen := map[string]bool{}
	out := []string{}
	for _, name := range normalizeSkillNames(names) {
		if allowed[name] && !seen[name] {
			seen[name] = true
			out = append(out, name)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func localSkillResolve(_ context.Context, request SkillResolveRequest) (SkillResolveDecision, error) {
	if strings.TrimSpace(request.Intent) == "" {
		return SkillResolveDecision{}, errors.New("intent is required")
	}
	words := strings.FieldsFunc(strings.ToLower(request.Intent), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	type scored struct {
		name  string
		score int
	}
	var matches []scored
	for _, candidate := range request.Candidates {
		text := strings.ToLower(candidate.Name + " " + candidate.Description)
		score := 0
		for _, word := range words {
			if len([]rune(word)) > 1 && strings.Contains(text, word) {
				score++
			}
		}
		if score > 0 {
			matches = append(matches, scored{candidate.Name, score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].name < matches[j].name
		}
		return matches[i].score > matches[j].score
	})
	max := request.MaxSkills
	if max <= 0 {
		max = 3
	}
	names := []string{}
	for _, match := range matches {
		names = append(names, match.name)
		if len(names) >= max {
			break
		}
	}
	return SkillResolveDecision{Names: names, Reason: "matched by local skill catalog search"}, nil
}
