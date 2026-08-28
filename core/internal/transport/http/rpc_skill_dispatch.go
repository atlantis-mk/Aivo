package http

import (
	"aivo/core/domain"
	"context"
	"encoding/json"
)

func (api *API) callSkillRPC(ctx context.Context, method string, args []json.RawMessage) (interface{}, bool, error) {
	switch method {
	case "ScanGlobalSkills":
		result, err := api.service.ScanGlobalSkills(ctx)
		return result, true, err
	case "ScanProjectSkills":
		input, err := arg[domain.SkillScanInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ScanProjectSkills(ctx, input)
		return result, true, err
	case "ListSkills":
		input, err := arg[domain.SkillListInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ListSkills(ctx, input)
		return result, true, err
	case "ImportSkill":
		input, err := arg[domain.SkillImportInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.ImportSkill(ctx, input)
		return result, true, err
	case "IgnoreSkillCandidatesByName":
		input, err := arg[domain.SkillIgnoreCandidatesInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.IgnoreSkillCandidatesByName(ctx, input)
		return result, true, err
	case "SetSkillEnabled":
		input, err := arg[domain.SkillEnabledInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetSkillEnabled(ctx, input)
		return result, true, err
	case "GetManagedSkillForEdit":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetManagedSkillForEdit(ctx, id)
		return result, true, err
	case "UpdateManagedSkill":
		input, err := arg[domain.SkillUpdateInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.UpdateManagedSkill(ctx, input)
		return result, true, err
	case "DeleteManagedSkill":
		id, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		err = api.service.DeleteManagedSkill(ctx, id)
		return map[string]bool{"ok": err == nil}, true, err
	case "LoadSkillIntoSession":
		input, err := arg[domain.LoadSkillIntoSessionInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.LoadSkillIntoSession(ctx, input)
		return result, true, err
	case "GetSessionActiveSkills":
		sessionID, err := arg[string](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.GetSessionActiveSkills(ctx, sessionID)
		return result, true, err
	case "SetSessionActiveSkills":
		input, err := arg[domain.SessionActiveSkillsInput](args, 0)
		if err != nil {
			return nil, true, err
		}
		result, err := api.service.SetSessionActiveSkills(ctx, input)
		return result, true, err
	default:
		return nil, false, nil
	}
}
