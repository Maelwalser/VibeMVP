package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vibe-menu/internal/manifest"
)

// ── Asset form fields ─────────────────────────────────────────────────────────

func newAssetForm(a manifest.AssetDef) []Field {
	usageIdx := 0
	if a.Usage == manifest.AssetUsageInspiration {
		usageIdx = 1
	}
	assetTypeIdx := 0
	assetTypeOpts := []string{"image", "icon", "font", "video", "mockup", "moodboard"}
	for i, opt := range assetTypeOpts {
		if opt == a.AssetType {
			assetTypeIdx = i
			break
		}
	}
	formatIdx := 0
	formatOpts := []string{"png", "jpg", "svg", "gif", "mp4", "pdf", "figma", "sketch", "other"}
	for i, opt := range formatOpts {
		if opt == a.Format {
			formatIdx = i
			break
		}
	}
	return []Field{
		{
			Key: "name", Label: "name          ", Kind: KindText,
			Value: a.Name,
		},
		{
			Key: "path", Label: "path          ", Kind: KindText,
			Value: a.Path,
		},
		{
			Key: "asset_type", Label: "asset_type    ", Kind: KindSelect,
			Options: assetTypeOpts, Value: assetTypeOpts[assetTypeIdx], SelIdx: assetTypeIdx,
		},
		{
			Key: "format", Label: "format        ", Kind: KindSelect,
			Options: formatOpts, Value: formatOpts[formatIdx], SelIdx: formatIdx,
		},
		{
			Key: "usage", Label: "usage         ", Kind: KindSelect,
			Options: []string{"project", "inspiration"}, Value: []string{"project", "inspiration"}[usageIdx], SelIdx: usageIdx,
		},
		{
			Key: "description", Label: "description   ", Kind: KindText,
			Value: a.Description,
		},
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) updateAssets(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.assetSubView == ceViewList {
		return fe.updateAssetList(key)
	}
	return fe.updateAssetForm(key)
}

func (fe FrontendEditor) updateAssetList(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	n := len(fe.assets)
	switch key.String() {
	case "j", "down":
		if n > 0 && fe.assetIdx < n-1 {
			fe.assetIdx++
		}
	case "k", "up":
		if fe.assetIdx > 0 {
			fe.assetIdx--
		}
	case "a":
		fe.assets = append(fe.assets, manifest.AssetDef{})
		fe.assetIdx = len(fe.assets) - 1
		fe.assetForm = newAssetForm(manifest.AssetDef{})
		fe.assetFormIdx = 0
		fe.assetSubView = ceViewForm
		return fe.tryEnterInsert()
	case "d":
		if n > 0 {
			fe.assets = append(fe.assets[:fe.assetIdx], fe.assets[fe.assetIdx+1:]...)
			if fe.assetIdx > 0 && fe.assetIdx >= len(fe.assets) {
				fe.assetIdx = len(fe.assets) - 1
			}
		}
	case "enter":
		if n > 0 {
			a := fe.assets[fe.assetIdx]
			fe.assetForm = newAssetForm(a)
			fe.assetFormIdx = 0
			fe.assetSubView = ceViewForm
		}
	}
	return fe, nil
}

func (fe FrontendEditor) updateAssetForm(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.dd.Open {
		return fe.updateAssetFormDropdown(key)
	}
	switch key.String() {
	case "j", "down":
		if fe.assetFormIdx < len(fe.assetForm)-1 {
			fe.assetFormIdx++
		}
	case "k", "up":
		if fe.assetFormIdx > 0 {
			fe.assetFormIdx--
		}
	case "enter", " ":
		f := &fe.assetForm[fe.assetFormIdx]
		if f.Kind == KindSelect || f.Kind == KindMultiSelect {
			fe.dd.Open = true
			if f.Kind == KindSelect {
				fe.dd.OptIdx = f.SelIdx
			} else {
				fe.dd.OptIdx = f.DDCursor
			}
		} else {
			return fe.tryEnterInsert()
		}
	case "H", "shift+left":
		f := &fe.assetForm[fe.assetFormIdx]
		if f.Kind == KindSelect {
			f.CyclePrev()
		}
	case "i", "a":
		if fe.assetForm[fe.assetFormIdx].CanEditAsText() {
			return fe.tryEnterInsert()
		}
	case "b", "esc":
		fe.saveAssetForm()
		fe.assetSubView = ceViewList
	}
	fe.saveAssetForm()
	return fe, nil
}

func (fe FrontendEditor) updateAssetFormDropdown(key tea.KeyMsg) (FrontendEditor, tea.Cmd) {
	if fe.assetFormIdx >= len(fe.assetForm) {
		fe.dd.Open = false
		return fe, nil
	}
	f := &fe.assetForm[fe.assetFormIdx]
	fe.dd.OptIdx = NavigateDropdown(key.String(), fe.dd.OptIdx, len(f.Options))
	switch key.String() {
	case " ":
		if f.Kind == KindMultiSelect {
			f.ToggleMultiSelect(fe.dd.OptIdx)
			f.DDCursor = fe.dd.OptIdx
		} else {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			fe.dd.Open = false
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
		}
	case "enter":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		} else {
			f.SelIdx = fe.dd.OptIdx
			if fe.dd.OptIdx < len(f.Options) {
				f.Value = f.Options[fe.dd.OptIdx]
			}
			if f.PrepareCustomEntry() {
				return fe.tryEnterInsert()
			}
		}
		fe.dd.Open = false
	case "esc", "b":
		if f.Kind == KindMultiSelect {
			f.DDCursor = fe.dd.OptIdx
		}
		fe.dd.Open = false
	}
	fe.saveAssetForm()
	return fe, nil
}

func (fe *FrontendEditor) saveAssetForm() {
	if fe.assetIdx >= len(fe.assets) {
		return
	}
	a := &fe.assets[fe.assetIdx]
	a.Name = fieldGet(fe.assetForm, "name")
	a.Path = fieldGet(fe.assetForm, "path")
	a.AssetType = fieldGet(fe.assetForm, "asset_type")
	a.Format = fieldGet(fe.assetForm, "format")
	a.Usage = manifest.AssetUsage(fieldGet(fe.assetForm, "usage"))
	a.Description = fieldGet(fe.assetForm, "description")
}

// ── View ──────────────────────────────────────────────────────────────────────

func (fe FrontendEditor) viewAssets(w int) []string {
	switch fe.assetSubView {
	case ceViewList:
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  # Assets — a: add  d: delete  Enter: edit"), "")
		if len(fe.assets) == 0 {
			lines = append(lines, StyleSectionDesc.Render("  (no assets yet — press 'a' to add)"))
		} else {
			for i, a := range fe.assets {
				name := a.Name
				if name == "" {
					name = fmt.Sprintf("(asset #%d)", i+1)
				}
				badge := string(a.Usage)
				if badge == "" {
					badge = "project"
				}
				lines = append(lines, renderListItem(w, i == fe.assetIdx, "  ▶ ", name, "["+badge+"] "+a.Path))
			}
		}
		return lines

	case ceViewForm:
		name := fieldGet(fe.assetForm, "name")
		if name == "" {
			name = "(new asset)"
		}
		var lines []string
		lines = append(lines, StyleSectionDesc.Render("  ← ")+StyleFieldKey.Render(name), "")
		lines = append(lines, renderFormFields(w, fe.assetForm, fe.assetFormIdx, fe.internalMode == ModeInsert, fe.formInput, fe.dd.Open, fe.dd.OptIdx)...)
		return lines
	}
	return nil
}
