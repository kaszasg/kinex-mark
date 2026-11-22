package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kaszasg/kinex-mark/internal/model"
)

// tagTestPath returns /api/tags/<id>.
func tagTestPath(id int64) string { return "/api/tags/" + strconv.FormatInt(id, 10) }

func TestTagCRUD(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleCustomer)

	res := c.get("/api/tags")
	assertJSON(t, expectStatus(t, res, http.StatusOK), `{"items":[]}`)

	created := decodeStatus[model.Tag](t, c.post("/api/tags", map[string]any{"name": "  Ügyfelek  "}), http.StatusCreated)
	if created.Name != "Ügyfelek" || created.Color != model.TagColorNeutral || created.LinkCount != 0 || !created.CreatedAt.Equal(e.clock.Now()) {
		t.Errorf("created = %+v", created)
	}
	blue := decodeStatus[model.Tag](t, c.post("/api/tags", map[string]any{"name": "Alfa", "color": "blue"}), http.StatusCreated)
	if blue.Color != model.TagColorBlue {
		t.Errorf("color = %s", blue.Color)
	}
	for _, color := range model.TagColors {
		tg := decodeStatus[model.Tag](t, c.post("/api/tags", map[string]any{"name": "Szín " + string(color), "color": color}), http.StatusCreated)
		if tg.Color != color {
			t.Errorf("color %s stored as %s", color, tg.Color)
		}
	}

	body := expectError(t, c.post("/api/tags", map[string]any{"name": "ügyfelek"}), http.StatusConflict, codeTagDuplicate)
	if id, _ := body.Details["tagId"].(float64); int64(id) != created.ID {
		t.Errorf("details.tagId = %v, want %d", body.Details["tagId"], created.ID)
	}

	list := decode[listResponse[model.Tag]](t, c.get("/api/tags")).Items
	if len(list) != 2+len(model.TagColors) || list[0].Name != "Alfa" {
		t.Errorf("list = %+v", list)
	}

	// Counts follow the links that carry the tag.
	l1 := linkTestCreate(t, c, map[string]any{"url": "https://t1.example/", "tagIds": []int64{blue.ID}})
	linkTestCreate(t, c, map[string]any{"url": "https://t2.example/", "tagIds": []int64{blue.ID}})
	expectStatus(t, c.post(linkTestPath(l1.ID, "/archive"), nil), http.StatusOK)
	list = decode[listResponse[model.Tag]](t, c.get("/api/tags")).Items
	if list[0].ID != blue.ID || list[0].LinkCount != 2 || list[0].ActiveLinkCount != 1 {
		t.Errorf("counts = %+v", list[0])
	}

	e.clock.Advance(time.Minute)
	renamed := decode[model.Tag](t, c.patch(tagTestPath(blue.ID), map[string]any{"name": "Béta"}))
	if renamed.Name != "Béta" || renamed.Color != model.TagColorBlue || !renamed.UpdatedAt.Equal(e.clock.Now()) {
		t.Errorf("renamed = %+v", renamed)
	}
	recolored := decode[model.Tag](t, c.patch(tagTestPath(blue.ID), map[string]any{"color": "rose"}))
	if recolored.Name != "Béta" || recolored.Color != model.TagColorRose {
		t.Errorf("recolored = %+v", recolored)
	}
	same := decode[model.Tag](t, c.patch(tagTestPath(blue.ID), map[string]any{}))
	if same.Name != "Béta" {
		t.Errorf("empty patch = %+v", same)
	}
	// Renaming to its own name in another case is not a duplicate.
	decode[model.Tag](t, c.patch(tagTestPath(blue.ID), map[string]any{"name": "BÉTA"}))
	expectError(t, c.patch(tagTestPath(blue.ID), map[string]any{"name": "ÜGYFELEK"}), http.StatusConflict, codeTagDuplicate)

	res = c.del(tagTestPath(blue.ID))
	expectStatus(t, res, http.StatusNoContent)
	if got := decode[LinkDetail](t, c.get(linkTestPath(l1.ID))); len(got.TagIDs) != 0 || len(got.Tags) != 0 {
		t.Errorf("deleted tag still on the link: %+v", got.Tags)
	}
	expectError(t, c.del(tagTestPath(blue.ID)), http.StatusNotFound, codeNotFound)
	expectError(t, c.patch(tagTestPath(blue.ID), map[string]any{"name": "X"}), http.StatusNotFound, codeNotFound)
	expectError(t, c.patch("/api/tags/x", map[string]any{}), http.StatusBadRequest, codeInvalidID)
	expectError(t, c.del("/api/tags/0"), http.StatusBadRequest, codeInvalidID)

	// Tags are personal: another user sees none of them.
	other, _ := linkTestLoginAs(t, e, model.RoleAdmin)
	assertJSON(t, expectStatus(t, other.get("/api/tags"), http.StatusOK), `{"items":[]}`)
	decodeStatus[model.Tag](t, other.post("/api/tags", map[string]any{"name": "Ügyfelek"}), http.StatusCreated)
}

func TestTagValidation(t *testing.T) {
	t.Parallel()
	e, faults := linkTestFaultEnv(t)
	c, _ := linkTestLoginAs(t, e, model.RoleDeveloper)
	tag := linkTestCreateTag(t, c, "Meglévő")

	cases := []struct {
		name string
		body map[string]any
		want map[string]string
	}{
		{"missing name", map[string]any{}, map[string]string{"name": "required"}},
		{"blank name", map[string]any{"name": "   "}, map[string]string{"name": "required"}},
		{"long name", map[string]any{"name": strings.Repeat("ő", 41)}, map[string]string{"name": "too_long"}},
		{"control", map[string]any{"name": "a\tb"}, map[string]string{"name": "invalid"}},
		{"bidi", map[string]any{"name": "a‏b"}, map[string]string{"name": "invalid"}},
		{"bad color", map[string]any{"name": "Jó", "color": "red"}, map[string]string{"color": "invalid"}},
		{"empty color", map[string]any{"name": "Jó", "color": ""}, map[string]string{"color": "invalid"}},
		{"status color", map[string]any{"name": "", "color": "primary"}, map[string]string{"name": "required", "color": "invalid"}},
	}
	for _, tc := range cases {
		t.Run("create "+tc.name, func(t *testing.T) {
			expectFields(t, c.post("/api/tags", tc.body), tc.want)
		})
	}
	decodeStatus[model.Tag](t, c.post("/api/tags", map[string]any{"name": strings.Repeat("ő", 40)}), http.StatusCreated)
	expectFields(t, c.patch(tagTestPath(tag.ID), map[string]any{"name": ""}), map[string]string{"name": "required"})
	expectFields(t, c.patch(tagTestPath(tag.ID), map[string]any{"color": "teal"}), map[string]string{"color": "invalid"})
	expectError(t, c.post("/api/tags", map[string]any{"name": "x", "linkCount": 3}), http.StatusBadRequest, codeInvalidJSON)
	expectError(t, c.patch(tagTestPath(tag.ID), "[]"), http.StatusBadRequest, codeInvalidJSON)

	faults.set("CreateTag", fmt.Errorf("insert tag: %w", model.ErrLimitReached))
	expectError(t, c.post("/api/tags", map[string]any{"name": "Túl sok"}), http.StatusConflict, codeTagLimitReached)
	faults.set("CreateTag", errors.New("injected"))
	expectError(t, c.post("/api/tags", map[string]any{"name": "Hiba"}), http.StatusInternalServerError, codeInternalError)
	faults.set("CreateTag", nil)
}

func TestTagLimitReal(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	c, u := linkTestLoginAs(t, e, model.RoleCustomer)
	for i := 0; i < model.MaxTagsPerUser; i++ {
		if _, err := e.store.CreateTag(e.ctx(), u.ID, fmt.Sprintf("tag-%03d", i), model.TagColorNeutral); err != nil {
			t.Fatal(err)
		}
	}
	expectError(t, c.post("/api/tags", map[string]any{"name": "one more"}), http.StatusConflict, codeTagLimitReached)
}
