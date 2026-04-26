package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mms/sleutel/internal/crypto"
	"github.com/mms/sleutel/internal/model"
	"github.com/mms/sleutel/internal/vault"
)

type screen int

const (
	screenUnlock screen = iota
	screenMain
	screenDetail
	screenAdd
	screenEdit
	screenQA
	screenQAForm
)

type vaultOpenedMsg struct {
	v   *vault.Vault
	err error
}

// App is the top-level bubbletea model. It owns the active screen and
// handles transitions between them.
type App struct {
	active    screen
	vaultPath string
	v         *vault.Vault
	unlock    unlockScreen
	main      MainScreen
	detail    detailScreen
	form      formScreen
	qa        qaScreen
	qaForm    qaFormScreen
}

func NewApp(vaultPath string) App {
	return App{
		active:    screenUnlock,
		vaultPath: vaultPath,
		unlock:    newUnlockScreen(),
	}
}

func (a App) Init() tea.Cmd {
	return a.unlock.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case vaultOpenedMsg:
		if msg.err != nil {
			a.unlock.setError(msg.err)
			a.unlock.setLoading(false)
			return a, nil
		}
		a.v = msg.v
		a.main = NewMainScreen(a.v.List())
		a.main.width = a.unlock.width
		a.main.height = a.unlock.height
		a.active = screenMain
		return a, nil

	case openDetailMsg:
		a.detail = newDetailScreen(msg.entry, a.main.width, a.main.height)
		a.active = screenDetail
		return a, nil

	case closeDetailMsg:
		a.active = screenMain
		return a, nil

	case openAddMsg:
		a.form = newAddFormScreen(a.main.width, a.main.height)
		a.active = screenAdd
		return a, a.form.Init()

	case openEditMsg:
		a.form = newEditFormScreen(msg.entry, a.detail.width, a.detail.height)
		a.active = screenEdit
		return a, a.form.Init()

	case submitAddMsg:
		if a.v != nil {
			added, err := a.v.Add(msg.entry)
			a.main = NewMainScreen(a.v.List())
			a.main.width = a.form.width
			a.main.height = a.form.height
			if err == nil {
				a.detail = newDetailScreen(added, a.form.width, a.form.height)
				a.active = screenDetail
				return a, nil
			}
		}
		a.active = screenMain
		return a, nil

	case submitAddOpenQAMsg:
		if a.v != nil {
			added, err := a.v.Add(msg.entry)
			a.main = NewMainScreen(a.v.List())
			a.main.width = a.form.width
			a.main.height = a.form.height
			if err == nil {
				a.qa = newQAScreen(added, a.form.width, a.form.height)
				a.active = screenQA
				return a, nil
			}
		}
		a.active = screenMain
		return a, nil

	case submitEditMsg:
		if a.v != nil {
			updated, err := a.v.Edit(msg.id, msg.entry)
			if err == nil {
				a.detail = newDetailScreen(updated, a.form.width, a.form.height)
				a.main = NewMainScreen(a.v.List())
				a.main.width = a.form.width
				a.main.height = a.form.height
			}
		}
		a.active = screenDetail
		return a, nil

	case openQAMsg:
		a.qa = newQAScreen(msg.entry, a.detail.width, a.detail.height)
		a.active = screenQA
		return a, nil

	case closeQAMsg:
		w, h := a.main.width, a.main.height
		a.main = NewMainScreen(a.v.List())
		a.main.width = w
		a.main.height = h
		a.detail = newDetailScreen(msg.entry, a.qa.width, a.qa.height)
		a.active = screenDetail
		return a, nil

	case openQAFormMsg:
		a.qaForm = newQAFormScreen(msg.entryID, msg.idx, msg.sq, a.qa.width, a.qa.height)
		a.active = screenQAForm
		return a, a.qaForm.Init()

	case submitQAFormMsg:
		if a.v != nil {
			var updated model.Entry
			var err error
			if msg.idx < 0 {
				updated, err = a.v.AddSecurityQuestion(msg.entryID, msg.sq)
			} else {
				updated, err = a.v.UpdateSecurityQuestion(msg.entryID, msg.idx, msg.sq)
			}
			if err == nil {
				a.qa.entry = updated
			}
		}
		a.active = screenQA
		return a, nil

	case deleteQAMsg:
		if a.v != nil {
			updated, err := a.v.DeleteSecurityQuestion(msg.entryID, msg.idx)
			if err == nil {
				a.qa.entry = updated
				if a.qa.cursor >= len(updated.SecurityQuestions) && a.qa.cursor > 0 {
					a.qa.cursor--
				}
			}
		}
		a.active = screenQA
		return a, nil

	case cancelQAFormMsg:
		a.active = screenQA
		return a, nil

	case deleteEntryMsg:
		if a.v != nil {
			a.v.Delete(msg.id)
			a.main = NewMainScreen(a.v.List())
			a.main.width = a.detail.width
			a.main.height = a.detail.height
		}
		a.active = screenMain
		return a, nil

	case cancelFormMsg:
		// Return to wherever made sense: detail if editing, main if adding.
		if a.active == screenEdit {
			a.active = screenDetail
		} else {
			a.active = screenMain
		}
		return a, nil

	case tea.WindowSizeMsg:
		a.unlock.width = msg.Width
		a.unlock.height = msg.Height
		a.main.width = msg.Width
		a.main.height = msg.Height
		a.detail.width = msg.Width
		a.detail.height = msg.Height
		a.form.width = msg.Width
		a.form.height = msg.Height
		a.qa.width = msg.Width
		a.qa.height = msg.Height
		a.qaForm.width = msg.Width
		a.qaForm.height = msg.Height
		return a, nil
	}

	switch a.active {
	case screenUnlock:
		next, cmd := a.unlock.Update(msg)
		ul := next.(unlockScreen)

		if ul.submitted {
			ul.submitted = false
			ul.setLoading(true)
			ul.setError(nil)
			pw := ul.password()
			a.unlock = ul
			return a, openVaultCmd(a.vaultPath, pw)
		}
		a.unlock = ul
		return a, cmd

	case screenMain:
		next, cmd := a.main.Update(msg)
		a.main = next.(MainScreen)
		return a, cmd

	case screenDetail:
		next, cmd := a.detail.Update(msg)
		a.detail = next.(detailScreen)
		return a, cmd

	case screenAdd, screenEdit:
		next, cmd := a.form.Update(msg)
		a.form = next.(formScreen)
		return a, cmd

	case screenQA:
		next, cmd := a.qa.Update(msg)
		a.qa = next.(qaScreen)
		return a, cmd

	case screenQAForm:
		next, cmd := a.qaForm.Update(msg)
		a.qaForm = next.(qaFormScreen)
		return a, cmd
	}

	return a, nil
}

func (a App) View() string {
	switch a.active {
	case screenUnlock:
		return a.unlock.View()
	case screenMain:
		return a.main.View()
	case screenDetail:
		return a.detail.View()
	case screenAdd, screenEdit:
		return a.form.View()
	case screenQA:
		return a.qa.View()
	case screenQAForm:
		return a.qaForm.View()
	}
	return ""
}

func openVaultCmd(path string, pw []byte) tea.Cmd {
	return func() tea.Msg {
		defer crypto.Zero(pw)
		v, err := vault.Open(path, pw)
		if err != nil {
			return vaultOpenedMsg{err: err}
		}
		return vaultOpenedMsg{v: v}
	}
}
