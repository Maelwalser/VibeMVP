package ui

import "github.com/vibe-menu/internal/manifest"

// ── Action form helpers ────────────────────────────────────────────────────────

// actionTypesForComponent returns the action_type options appropriate for a given component type.
func actionTypesForComponent(compType string) []string {
	switch compType {
	case "Form":
		return []string{"Submit Form", "Fetch Data", "Reset Form", "Navigate", "Show Toast", "Update State", "Open Modal", "Custom"}
	case "Table":
		return []string{"Fetch Data", "Navigate", "Delete", "Refresh", "Export", "Show Toast", "Update State", "Open Modal", "Custom"}
	case "Card":
		return []string{"Navigate", "Fetch Data", "Open Modal", "Show Toast", "Update State", "Custom"}
	case "List":
		return []string{"Navigate", "Fetch Data", "Delete", "Refresh", "Open Modal", "Show Toast", "Update State", "Custom"}
	case "Chart":
		return []string{"Fetch Data", "Update State", "Download", "Custom"}
	case "Modal":
		return []string{"Submit Form", "Close Modal", "Navigate", "Fetch Data", "Show Toast", "Update State", "Custom"}
	case "Button":
		return []string{"Navigate", "Submit Form", "Open Modal", "Close Modal", "Show Toast", "Update State", "Download", "Upload", "Custom"}
	case "Navigation":
		return []string{"Navigate", "Custom"}
	default:
		return []string{"Fetch Data", "Submit Form", "Navigate", "Show Toast", "Update State", "Open Modal", "Close Modal", "Reset Form", "Download", "Upload", "Delete", "Refresh", "Export", "Custom"}
	}
}

// actionTypeVisibleExtras maps each action_type to the set of conditional
// form fields that should be visible for that type.
var actionTypeVisibleExtras = map[string]map[string]bool{
	"Submit Form":  {"form_target": true, "endpoint": true, "success_action": true, "error_action": true},
	"Fetch Data":   {"endpoint": true, "success_action": true, "error_action": true},
	"Navigate":     {"target_page": true},
	"Show Toast":   {"toast_message": true, "toast_type": true},
	"Open Modal":   {"modal_target": true},
	"Close Modal":  {"modal_target": true},
	"Update State": {"state_key": true, "state_value": true},
	"Reset Form":   {"form_target": true},
	"Download":     {"endpoint": true, "success_action": true, "error_action": true},
	"Upload":       {"endpoint": true, "success_action": true, "error_action": true},
	"Delete":       {"endpoint": true, "confirm_dialog": true, "success_action": true, "error_action": true},
	"Refresh":      {"endpoint": true, "success_action": true, "error_action": true},
	"Export":       {"endpoint": true, "success_action": true, "error_action": true},
	"Custom":       {"custom_handler": true},
}

// alwaysVisibleActionKeys are shown for every action type.
var alwaysVisibleActionKeys = map[string]bool{
	"trigger": true, "action_type": true, "description": true,
}

// defaultHttpMethod returns a sensible HTTP method default for a given action type.
func defaultHttpMethod(actionType string) string {
	switch actionType {
	case "Submit Form", "Upload":
		return "POST"
	case "Delete":
		return "DELETE"
	default:
		return "GET"
	}
}

// isActionFieldHidden returns true when the given action form field should be
// hidden based on the current action_type selection.
func isActionFieldHidden(fields []Field, idx int) bool {
	key := fields[idx].Key
	if alwaysVisibleActionKeys[key] {
		return false
	}
	actionType := ""
	for _, f := range fields {
		if f.Key == "action_type" {
			actionType = f.DisplayValue()
			break
		}
	}
	extras := actionTypeVisibleExtras[actionType]
	return !extras[key]
}

// actionVisibleFields returns only the action form fields that should be rendered.
func actionVisibleFields(fields []Field) []Field {
	out := make([]Field, 0, len(fields))
	for i := range fields {
		if !isActionFieldHidden(fields, i) {
			out = append(out, fields[i])
		}
	}
	return out
}

// actionVisibleIdx maps a full-list form index to its position within the visible list.
func actionVisibleIdx(fields []Field, fullIdx int) int {
	vis := 0
	for i := range fullIdx {
		if !isActionFieldHidden(fields, i) {
			vis++
		}
	}
	return vis
}

// nextActionFormIdx advances the action form cursor, skipping hidden fields.
func nextActionFormIdx(fields []Field, cur int) int {
	return nextFormIdx(fields, cur, isActionFieldHidden)
}

// prevActionFormIdx retreats the action form cursor, skipping hidden fields.
func prevActionFormIdx(fields []Field, cur int) int {
	return prevFormIdx(fields, cur, isActionFieldHidden)
}

// restoreActionForm populates form fields from a saved ComponentActionDef.
func (fe *FrontendEditor) restoreActionForm(a manifest.ComponentActionDef) {
	if a.Trigger != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "trigger", a.Trigger)
	}
	if a.ActionType != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "action_type", a.ActionType)
	}
	ep := a.Endpoint
	if ep == "" {
		ep = "None"
	}
	fe.actionForm = setFieldValue(fe.actionForm, "endpoint", ep)
	if a.HttpMethod != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "http_method", a.HttpMethod)
	}
	if a.RequestBody != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "request_body", a.RequestBody)
	}
	if a.SuccessAction != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "success_action", a.SuccessAction)
	}
	if a.ErrorAction != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "error_action", a.ErrorAction)
	}
	ft := a.FormTarget
	if ft == "" {
		ft = "(none)"
	}
	fe.actionForm = setFieldValue(fe.actionForm, "form_target", ft)
	mt := a.ModalTarget
	if mt == "" {
		mt = "(none)"
	}
	fe.actionForm = setFieldValue(fe.actionForm, "modal_target", mt)
	tp := a.TargetPage
	if tp == "" {
		tp = "(none)"
	}
	fe.actionForm = setFieldValue(fe.actionForm, "target_page", tp)
	if a.ToastMessage != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "toast_message", a.ToastMessage)
	}
	if a.ToastType != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "toast_type", a.ToastType)
	}
	if a.ConfirmDialog != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "confirm_dialog", a.ConfirmDialog)
	}
	if a.StateKey != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "state_key", a.StateKey)
	}
	if a.StateValue != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "state_value", a.StateValue)
	}
	if a.CustomHandler != "" {
		fe.actionForm = setFieldValue(fe.actionForm, "custom_handler", a.CustomHandler)
	}
	fe.actionForm = setFieldValue(fe.actionForm, "description", a.Description)
}

// defaultActionFormFields builds the full action form field list.
// All conditional fields are always present; isActionFieldHidden governs visibility.
func defaultActionFormFields(compType string, endpointOptions, pageRoutes, formComponents, modalComponents []string) []Field {
	endpointWithNone := append([]string{"None"}, endpointOptions...)
	pageWithNone := append([]string{"(none)"}, pageRoutes...)
	formWithNone := append([]string{"(none)"}, formComponents...)
	modalWithNone := append([]string{"(none)"}, modalComponents...)
	actionTypes := actionTypesForComponent(compType)
	defaultAction := ""
	if len(actionTypes) > 0 {
		defaultAction = actionTypes[0]
	}
	return []Field{
		{
			Key: "trigger", Label: "trigger       ", Kind: KindSelect,
			Options: []string{"onClick", "onSubmit", "onLoad", "onMount", "onChange", "onHover", "onScroll", "onKeyPress", "Custom"},
			Value:   "onClick",
		},
		{
			Key: "action_type", Label: "action_type   ", Kind: KindSelect,
			Options: actionTypes,
			Value:   defaultAction,
		},
		// Form component target — Submit Form, Reset Form
		{
			Key: "form_target", Label: "form_target   ", Kind: KindSelect,
			Options: formWithNone,
			Value:   "(none)",
		},
		// API request fields — Fetch Data, Submit Form, Download, Upload, Delete, Refresh, Export
		{
			Key: "endpoint", Label: "endpoint      ", Kind: KindSelect,
			Options: endpointWithNone,
			Value:   "None",
		},
		{
			Key: "http_method", Label: "http_method   ", Kind: KindSelect,
			Options: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			Value:   defaultHttpMethod(defaultAction),
		},
		{
			Key: "request_body", Label: "request_body  ", Kind: KindSelect,
			Options: []string{"JSON", "FormData", "Multipart", "Raw", "None"},
			Value:   "JSON",
		},
		{
			Key: "success_action", Label: "success_action", Kind: KindSelect,
			Options: []string{"None", "Show Toast", "Navigate", "Update State", "Refresh"},
			Value:   "None",
		},
		{
			Key: "error_action", Label: "error_action  ", Kind: KindSelect,
			Options: []string{"Show Toast", "Do Nothing", "Retry", "Navigate"},
			Value:   "Show Toast",
		},
		// Modal target — Open Modal, Close Modal
		{
			Key: "modal_target", Label: "modal_target  ", Kind: KindSelect,
			Options: modalWithNone,
			Value:   "(none)",
		},
		// Navigation
		{
			Key: "target_page", Label: "target_page   ", Kind: KindSelect,
			Options: pageWithNone,
			Value:   placeholderFor(pageRoutes, "(no pages configured)"),
		},
		// Toast
		{Key: "toast_message", Label: "toast_message ", Kind: KindText},
		{
			Key: "toast_type", Label: "toast_type    ", Kind: KindSelect,
			Options: []string{"success", "error", "info", "warning"},
			Value:   "success",
		},
		// Delete confirmation
		{
			Key: "confirm_dialog", Label: "confirm_dialog", Kind: KindSelect,
			Options: []string{"Yes", "No"},
			Value:   "Yes",
		},
		// State management
		{Key: "state_key", Label: "state_key     ", Kind: KindText},
		{Key: "state_value", Label: "state_value   ", Kind: KindText},
		// Custom handler
		{Key: "custom_handler", Label: "custom_handler", Kind: KindText},
		{Key: "description", Label: "description   ", Kind: KindText},
	}
}
