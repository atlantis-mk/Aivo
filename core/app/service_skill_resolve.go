package app

import (
	"context"
	"encoding/json"
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

func (s *Service) resolveSkillsWithAuxiliaryModel(ctx context.Context, request SkillResolveRequest) (SkillResolveDecision, error) {
	if len(request.Candidates) == 0 {
		return SkillResolveDecision{}, nil
	}
	catalog := make([]map[string]any, 0, len(request.Candidates))
	for _, candidate := range request.Candidates {
		catalog = append(catalog, map[string]any{"name": candidate.Name, "description": bounded(candidate.Description, 600), "scope": candidate.Scope, "source": candidate.Source, "status": candidate.Status})
	}
	payload, _ := json.MarshalIndent(map[string]any{"intent": request.Intent, "maxSkills": request.MaxSkills, "agentMode": request.AgentMode, "catalog": catalog}, "", "  ")
	messages := []domain.ChatMessage{
		{Role: "system", Text: "Select skills only from the provided catalog that directly help with the user's intent. Return strict JSON: {\"skills\":[\"exact-skill-name\"],\"reason\":\"short reason\"}. Do not invent names or select merely adjacent skills. Return an empty skills array when none clearly match."},
		{Role: "user", Text: string(payload)},
	}
	var lastErr error
	for _, model := range s.resolveAuxiliaryModels(ctx, nil) {
		reply, _, err := s.GenerateChatReply(ctx, messages, &model, "low", "default")
		if err != nil {
			lastErr = err
			continue
		}
		decision, err := parseSkillResolveDecision(reply)
		if err != nil {
			lastErr = err
			continue
		}
		return decision, nil
	}
	if lastErr != nil {
		return SkillResolveDecision{}, lastErr
	}
	return localSkillResolve(ctx, request)
}

func parseSkillResolveDecision(raw string) (SkillResolveDecision, error) {
	text := strings.TrimSpace(stripThinkBlocks(raw))
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	if start, end := strings.Index(text, "{"), strings.LastIndex(text, "}"); start >= 0 && end >= start {
		text = text[start : end+1]
	}
	var decoded struct {
		Skills []string `json:"skills"`
		Names  []string `json:"names"`
		Reason string   `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &decoded); err != nil {
		return SkillResolveDecision{}, err
	}
	names := decoded.Skills
	if len(names) == 0 {
		names = decoded.Names
	}
	return SkillResolveDecision{Names: normalizeSkillNames(names), Reason: strings.TrimSpace(decoded.Reason)}, nil
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
