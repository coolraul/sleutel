package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mms/sleutel/internal/crypto"
	"github.com/mms/sleutel/internal/vault"
)

type screen int

const (
	screenUnlock screen = iota
	screenMain
	screenDetail
	screenAdd
	screenEdit
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
			a.v.Add(msg.entry)
			a.main = NewMainScreen(a.v.List())
			a.main.width = a.form.width
			a.main.height = a.form.height
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
