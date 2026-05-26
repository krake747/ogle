package hoverlist_test

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	titleAlpha = "alpha"
	titleBeta  = "beta"
	testWidth  = 30
	testHeight = 20
)

type testItem struct {
	title string
	desc  string
}

func (i testItem) Title() string       { return i.title }
func (i testItem) Description() string { return i.desc }
func (i testItem) FilterValue() string { return i.title }

func TestNewDelegate(t *testing.T) {
	t.Parallel()

	base := list.NewDefaultDelegate()
	base.ShowDescription = false
	th := theme.Default()
	zm := zone.New()
	items := []list.Item{
		testItem{title: titleAlpha},
		testItem{title: titleBeta},
	}
	d := hoverlist.NewDelegate(base, th, zm)
	m := list.New(items, d, testWidth, testHeight)

	require.NotNil(t, d)
	require.NotNil(t, m)
}

func TestSetHover(t *testing.T) {
	t.Parallel()

	th := theme.DefaultLight()
	zm := zone.New()
	base := list.NewDefaultDelegate()
	base.ShowDescription = false
	items := []list.Item{
		testItem{title: titleAlpha},
		testItem{title: titleBeta},
	}
	d := hoverlist.NewDelegate(base, th, zm)
	m := list.New(items, d, testWidth, testHeight)

	var normal strings.Builder
	d.Render(&normal, m, 1, items[1])

	d.SetHover(1)

	var hovered strings.Builder
	d.Render(&hovered, m, 1, items[1])

	assert.NotEqual(t, normal.String(), hovered.String(),
		"hover background should differ from normal background")
}

func TestSetTheme(t *testing.T) {
	t.Parallel()

	zm := zone.New()
	base := list.NewDefaultDelegate()
	base.ShowDescription = false
	items := []list.Item{
		testItem{title: titleAlpha},
	}
	d := hoverlist.NewDelegate(base, theme.Default(), zm)
	m := list.New(items, d, testWidth, testHeight)

	var before strings.Builder
	d.Render(&before, m, 1, items[0])

	d.SetTheme(theme.DefaultLight())

	var after strings.Builder
	d.Render(&after, m, 1, items[0])

	assert.NotEqual(t, before.String(), after.String(),
		"different themes should produce different colour rendering")
}

func TestRender(t *testing.T) {
	t.Parallel()

	th := theme.DefaultLight()

	type testCase struct {
		name   string
		setup  func(d hoverlist.Delegate)
		index  int
		assert func(t *testing.T, rendered string, zm *zone.Manager)
	}

	cases := []testCase{
		{
			name:  "normal and selected items render differently",
			index: 1,
			assert: func(t *testing.T, rendered string, _ *zone.Manager) {
				t.Helper()

				d, m, items, _ := newRenderFixture(t, th)

				var selectedBuf strings.Builder
				d.Render(&selectedBuf, m, 0, items[0])

				assert.NotEqual(t, selectedBuf.String(), rendered,
					"selected and normal items should render differently")
			},
		},
		{
			name: "hover background differs from normal",
			setup: func(d hoverlist.Delegate) {
				d.SetHover(0)
			},
			index: 0,
			assert: func(t *testing.T, rendered string, _ *zone.Manager) {
				t.Helper()

				d, m, items, _ := newRenderFixture(t, th)

				var normalBuf strings.Builder
				d.Render(&normalBuf, m, 0, items[0])

				assert.NotEqual(t, normalBuf.String(), rendered,
					"hovered item should differ from non-hovered item")
			},
		},
		{
			name:  "produces bubblezone-marked output",
			index: 0,
			assert: func(t *testing.T, rendered string, zm *zone.Manager) {
				t.Helper()

				zm.Scan(rendered)

				require.Eventually(t, func() bool {
					zi := zm.Get("item-0")

					return zi != nil && !zi.IsZero()
				}, time.Second, 10*time.Millisecond)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d, m, items, zm := newRenderFixture(t, th)

			if tc.setup != nil {
				tc.setup(d)
			}

			var buf strings.Builder
			d.Render(&buf, m, tc.index, items[tc.index])

			tc.assert(t, buf.String(), zm)
		})
	}
}

func newRenderFixture(t *testing.T, th *theme.Theme) (
	hoverlist.Delegate, list.Model, []list.Item, *zone.Manager,
) {
	t.Helper()

	base := list.NewDefaultDelegate()
	base.ShowDescription = false
	zm := zone.New()
	items := []list.Item{
		testItem{title: titleAlpha},
		testItem{title: titleBeta},
	}
	d := hoverlist.NewDelegate(base, th, zm)
	m := list.New(items, d, testWidth, testHeight)

	return d, m, items, zm
}
