package service

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"mainstory-digital-library-takehome/internal/domain"
	"mainstory-digital-library-takehome/internal/repository"
)

type fakeBookRepo struct {
	byID map[uuid.UUID]*domain.Book
	list []domain.Book
}

func (f *fakeBookRepo) Create(ctx context.Context, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error) {
	b := &domain.Book{
		ID: uuid.New(), Title: title, Description: description, Author: author, Genre: genre,
		IsFiction: isFiction, Language: language, PriceCents: priceCents, Content: content,
		AddedAt: time.Now(),
	}
	if publishedDate != nil {
		if t, ok := publishedDate.(*time.Time); ok && t != nil {
			b.PublishedDate = t
		}
	}
	f.byID[b.ID] = b
	f.list = append(f.list, *b)
	return cloneBookPtr(b), nil
}

func cloneBookPtr(b *domain.Book) *domain.Book {
	c := *b
	return &c
}

func (f *fakeBookRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Book, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneBookPtr(b), nil
}

func (f *fakeBookRepo) ListCatalog(ctx context.Context, filter domain.BookListFilter, limit, offset int32) ([]domain.Book, error) {
	var matched []domain.Book
	for _, b := range f.list {
		bk := b
		bk.Content = ""
		if filter.Q != "" {
			q := strings.ToLower(filter.Q)
			if !strings.Contains(strings.ToLower(bk.Title), q) &&
				!strings.Contains(strings.ToLower(bk.Author), q) &&
				!strings.Contains(strings.ToLower(bk.Genre), q) {
				continue
			}
		}
		if filter.Title != "" && !strings.Contains(strings.ToLower(bk.Title), strings.ToLower(filter.Title)) {
			continue
		}
		if filter.Author != "" && !strings.Contains(strings.ToLower(bk.Author), strings.ToLower(filter.Author)) {
			continue
		}
		if filter.Genre != "" && !strings.Contains(strings.ToLower(bk.Genre), strings.ToLower(filter.Genre)) {
			continue
		}
		if filter.Language != "" && strings.ToLower(bk.Language) != strings.ToLower(filter.Language) {
			continue
		}
		if filter.IsFiction != nil && bk.IsFiction != *filter.IsFiction {
			continue
		}
		if filter.MinPriceCents != nil && bk.PriceCents < *filter.MinPriceCents {
			continue
		}
		if filter.MaxPriceCents != nil && bk.PriceCents > *filter.MaxPriceCents {
			continue
		}
		matched = append(matched, bk)
	}
	if int(offset) >= len(matched) {
		return nil, nil
	}
	end := int(offset) + int(limit)
	if end > len(matched) {
		end = len(matched)
	}
	out := make([]domain.Book, end-int(offset))
	copy(out, matched[int(offset):end])
	return out, nil
}

func (f *fakeBookRepo) ListRecentCatalogTop5(ctx context.Context) ([]domain.Book, error) {
	books := append([]domain.Book(nil), f.list...)
	sort.Slice(books, func(i, j int) bool {
		return books[i].AddedAt.After(books[j].AddedAt)
	})
	n := 5
	if len(books) < n {
		n = len(books)
	}
	out := make([]domain.Book, 0, n)
	for i := 0; i < n; i++ {
		bk := books[i]
		bk.Content = ""
		out = append(out, bk)
	}
	return out, nil
}

func (f *fakeBookRepo) GetCatalogByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Book, error) {
	var out []domain.Book
	for _, id := range ids {
		if b, ok := f.byID[id]; ok {
			c := *b
			c.Content = ""
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeBookRepo) Update(ctx context.Context, id uuid.UUID, title, description, author, genre string, isFiction bool, publishedDate interface{}, language string, priceCents int32, content string) (*domain.Book, error) {
	b, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	b.Title = title
	b.Description = description
	b.Author = author
	b.Genre = genre
	b.IsFiction = isFiction
	b.Language = language
	b.PriceCents = priceCents
	b.Content = content
	if publishedDate != nil {
		if t, ok := publishedDate.(*time.Time); ok {
			b.PublishedDate = t
		}
	}
	return cloneBookPtr(b), nil
}

func (f *fakeBookRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := f.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.byID, id)
	return nil
}

type fakeEntRepo struct {
	subs      map[uuid.UUID]bool
	purchase  map[string]bool
	books     map[uuid.UUID]bool
	entByID   map[uuid.UUID]*domain.Entitlement
	entByUser map[uuid.UUID][]domain.Entitlement
	allEnt    []domain.Entitlement
}

func keyPurchase(uid, bid uuid.UUID) string {
	return uid.String() + ":" + bid.String()
}

func (f *fakeEntRepo) Create(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, typ, status string, endsAt *time.Time, renewedAt *time.Time) (*domain.Entitlement, error) {
	e := &domain.Entitlement{
		ID: uuid.New(), UserID: userID, BookID: bookID, Type: typ, Status: status,
		EndsAt: endsAt, RenewedAt: renewedAt, CreatedAt: time.Now(),
	}
	f.entByID[e.ID] = e
	f.allEnt = append(f.allEnt, *e)
	f.entByUser[userID] = append(f.entByUser[userID], *e)
	f.syncSubs(userID)
	if typ == domain.EntitlementSinglePurchase && bookID != nil && status == domain.EntitlementActive {
		f.purchase[keyPurchase(userID, *bookID)] = true
	}
	return cloneEntPtr(e), nil
}

func (f *fakeEntRepo) syncSubs(userID uuid.UUID) {
	active := false
	for _, e := range f.entByID {
		if e.UserID == userID && e.Type == domain.EntitlementSubscription && e.Status == domain.EntitlementActive &&
			e.EndsAt != nil && e.EndsAt.After(time.Now()) {
			active = true
			break
		}
	}
	if active {
		f.subs[userID] = true
	} else {
		delete(f.subs, userID)
	}
}

func (f *fakeEntRepo) ExpireStaleSubscriptionsForUser(ctx context.Context, userID uuid.UUID) error {
	for _, e := range f.entByID {
		if e.UserID != userID || e.Type != domain.EntitlementSubscription || e.Status != domain.EntitlementActive {
			continue
		}
		if e.EndsAt != nil && !e.EndsAt.After(time.Now()) {
			e.Status = domain.EntitlementCancelled
		}
	}
	f.syncSubs(userID)
	return nil
}

func (f *fakeEntRepo) SetSubscriptionCancelledAt(ctx context.Context, id uuid.UUID, at time.Time) (*domain.Entitlement, error) {
	e, ok := f.entByID[id]
	if !ok || e.Type != domain.EntitlementSubscription || e.Status != domain.EntitlementActive {
		return nil, domain.ErrNotFound
	}
	t := at
	e.CancelledAt = &t
	f.syncSubs(e.UserID)
	return cloneEntPtr(e), nil
}

func cloneEntPtr(e *domain.Entitlement) *domain.Entitlement {
	c := *e
	return &c
}

func (f *fakeEntRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Entitlement, error) {
	e, ok := f.entByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneEntPtr(e), nil
}

func (f *fakeEntRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Entitlement, error) {
	_ = f.ExpireStaleSubscriptionsForUser(ctx, userID)
	list := f.entByUser[userID]
	if int(offset) >= len(list) {
		return nil, nil
	}
	end := int(offset) + int(limit)
	if end > len(list) {
		end = len(list)
	}
	out := make([]domain.Entitlement, end-int(offset))
	copy(out, list[int(offset):end])
	return out, nil
}

func (f *fakeEntRepo) ListAll(ctx context.Context, limit, offset int32) ([]domain.Entitlement, error) {
	if int(offset) >= len(f.allEnt) {
		return nil, nil
	}
	end := int(offset) + int(limit)
	if end > len(f.allEnt) {
		end = len(f.allEnt)
	}
	out := make([]domain.Entitlement, end-int(offset))
	copy(out, f.allEnt[int(offset):end])
	return out, nil
}

func (f *fakeEntRepo) Update(ctx context.Context, id uuid.UUID, status *string, endsAt *time.Time) (*domain.Entitlement, error) {
	e, ok := f.entByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if status != nil {
		e.Status = *status
		if e.Type == domain.EntitlementSubscription {
			if e.Status == domain.EntitlementActive {
				f.subs[e.UserID] = true
			} else {
				delete(f.subs, e.UserID)
			}
		}
	}
	if endsAt != nil {
		e.EndsAt = endsAt
	}
	f.syncSubs(e.UserID)
	return cloneEntPtr(e), nil
}

func (f *fakeEntRepo) HasActiveSubscription(ctx context.Context, userID uuid.UUID) (bool, error) {
	if err := f.ExpireStaleSubscriptionsForUser(ctx, userID); err != nil {
		return false, err
	}
	return f.subs[userID], nil
}

func (f *fakeEntRepo) HasActivePurchase(ctx context.Context, userID, bookID uuid.UUID) (bool, error) {
	return f.purchase[keyPurchase(userID, bookID)], nil
}

func (f *fakeEntRepo) GetActiveSubscriptionEntitlement(ctx context.Context, userID uuid.UUID) (*domain.Entitlement, error) {
	if err := f.ExpireStaleSubscriptionsForUser(ctx, userID); err != nil {
		return nil, err
	}
	for _, e := range f.entByID {
		if e.UserID == userID && e.Type == domain.EntitlementSubscription && e.Status == domain.EntitlementActive &&
			e.EndsAt != nil && e.EndsAt.After(time.Now()) {
			return cloneEntPtr(e), nil
		}
	}
	return nil, nil
}

func (f *fakeEntRepo) ListActivePurchasesByUser(ctx context.Context, userID uuid.UUID) ([]domain.Entitlement, error) {
	var out []domain.Entitlement
	for _, e := range f.entByUser[userID] {
		if e.Type == domain.EntitlementSinglePurchase && e.Status == domain.EntitlementActive && e.BookID != nil {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeEntRepo) BookExists(ctx context.Context, bookID uuid.UUID) (bool, error) {
	return f.books[bookID], nil
}

var (
	_ repository.BookStore        = (*fakeBookRepo)(nil)
	_ repository.EntitlementStore = (*fakeEntRepo)(nil)
)

func newFakeCatalog() (*fakeBookRepo, *fakeEntRepo) {
	return &fakeBookRepo{byID: make(map[uuid.UUID]*domain.Book)},
		&fakeEntRepo{
			subs: make(map[uuid.UUID]bool), purchase: make(map[string]bool),
			books: make(map[uuid.UUID]bool), entByID: make(map[uuid.UUID]*domain.Entitlement),
			entByUser: make(map[uuid.UUID][]domain.Entitlement),
		}
}

func TestBookService_MemberList_SubscriptionGrantsAccess(t *testing.T) {
	br, er := newFakeCatalog()
	u := uuid.New()
	bid := uuid.New()
	br.byID[bid] = &domain.Book{ID: bid, Title: "T", Language: "en", PriceCents: 100, AddedAt: time.Now()}
	br.list = []domain.Book{*br.byID[bid]}
	now := time.Now()
	end := now.AddDate(0, 0, domain.SubscriptionPeriodDays)
	_, _ = er.Create(context.Background(), u, nil, domain.EntitlementSubscription, domain.EntitlementActive, &end, &now)

	svc := NewBookService(br, er)
	items, err := svc.List(context.Background(), u, domain.RoleMember, domain.BookListFilter{}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].IsAccessible || items[0].AccessReason != domain.AccessReasonSubscription {
		t.Fatalf("%+v", items)
	}
}

func TestBookService_MemberGet_LockedStripsContent(t *testing.T) {
	br, er := newFakeCatalog()
	u := uuid.New()
	bid := uuid.New()
	br.byID[bid] = &domain.Book{ID: bid, Title: "Secret", Content: "hidden body", Language: "en", PriceCents: 1, AddedAt: time.Now()}
	br.list = []domain.Book{*br.byID[bid]}
	er.books[bid] = true

	svc := NewBookService(br, er)
	item, err := svc.Get(context.Background(), u, domain.RoleMember, bid)
	if err != nil {
		t.Fatal(err)
	}
	if item.Content != "" || item.IsAccessible || item.AccessReason != domain.AccessReasonLocked {
		t.Fatalf("%+v", item)
	}
}

func TestBookService_MemberGet_PurchaseShowsContent(t *testing.T) {
	br, er := newFakeCatalog()
	u := uuid.New()
	bid := uuid.New()
	br.byID[bid] = &domain.Book{ID: bid, Title: "B", Content: "full text", Language: "en", PriceCents: 1, AddedAt: time.Now()}
	er.books[bid] = true
	er.purchase[keyPurchase(u, bid)] = true

	svc := NewBookService(br, er)
	item, err := svc.Get(context.Background(), u, domain.RoleMember, bid)
	if err != nil {
		t.Fatal(err)
	}
	if item.Content != "full text" || !item.IsAccessible || item.AccessReason != domain.AccessReasonPurchased {
		t.Fatalf("%+v", item)
	}
}

func TestEntitlementService_MemberCannotCreateAsOther(t *testing.T) {
	_, er := newFakeCatalog()
	svc := NewEntitlementService(er)
	u := uuid.New()
	other := uuid.New()
	bid := uuid.New()
	er.books[bid] = true
	_, err := svc.Create(context.Background(), u, domain.RoleMember, CreateEntitlementInput{
		TargetUserID: &other,
		Type:         domain.EntitlementSinglePurchase,
		BookID:       &bid,
	})
	if err != domain.ErrForbidden {
		t.Fatalf("got %v", err)
	}
}

func TestEntitlementService_LibrarianCannotCreate(t *testing.T) {
	_, er := newFakeCatalog()
	svc := NewEntitlementService(er)
	u := uuid.New()
	bid := uuid.New()
	er.books[bid] = true
	_, err := svc.Create(context.Background(), u, domain.RoleLibrarian, CreateEntitlementInput{
		Type:   domain.EntitlementSinglePurchase,
		BookID: &bid,
	})
	if err != domain.ErrForbidden {
		t.Fatalf("got %v", err)
	}
}

func TestEntitlementService_MemberCreatesPurchase(t *testing.T) {
	_, er := newFakeCatalog()
	svc := NewEntitlementService(er)
	u := uuid.New()
	bid := uuid.New()
	er.books[bid] = true
	e, err := svc.Create(context.Background(), u, domain.RoleMember, CreateEntitlementInput{
		Type:   domain.EntitlementSinglePurchase,
		BookID: &bid,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.UserID != u || e.Type != domain.EntitlementSinglePurchase {
		t.Fatalf("%+v", e)
	}
}

func TestEntitlementService_AdminRequiresTargetUser(t *testing.T) {
	_, er := newFakeCatalog()
	svc := NewEntitlementService(er)
	bid := uuid.New()
	er.books[bid] = true
	_, err := svc.Create(context.Background(), uuid.New(), domain.RoleAdmin, CreateEntitlementInput{
		Type:   domain.EntitlementSinglePurchase,
		BookID: &bid,
	})
	if err != domain.ErrInvalidEntitlementRequest {
		t.Fatalf("got %v", err)
	}
}

func TestEntitlementService_MemberCannotReadOthersEntitlement(t *testing.T) {
	_, er := newFakeCatalog()
	svc := NewEntitlementService(er)
	u := uuid.New()
	other := uuid.New()
	eid := uuid.New()
	rn := time.Now()
	en := rn.AddDate(0, 0, domain.SubscriptionPeriodDays)
	er.entByID[eid] = &domain.Entitlement{ID: eid, UserID: other, Type: domain.EntitlementSubscription, Status: domain.EntitlementActive, RenewedAt: &rn, EndsAt: &en, CreatedAt: time.Now()}

	_, err := svc.Get(context.Background(), u, domain.RoleMember, eid)
	if err != domain.ErrForbidden {
		t.Fatalf("got %v", err)
	}
}

func TestValidateBookListFilter_ShortSearch(t *testing.T) {
	f := domain.BookListFilter{Q: "a"}
	if err := ValidateBookListFilter(f); err != domain.ErrSearchTermTooShort {
		t.Fatalf("got %v", err)
	}
}

func TestBookService_GuestList_AllLocked(t *testing.T) {
	br, er := newFakeCatalog()
	bid := uuid.New()
	br.byID[bid] = &domain.Book{ID: bid, Title: "Pub", Language: "en", PriceCents: 1, AddedAt: time.Now()}
	br.list = []domain.Book{*br.byID[bid]}

	svc := NewBookService(br, er)
	items, err := svc.List(context.Background(), uuid.Nil, "", domain.BookListFilter{}, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].IsAccessible || items[0].AccessReason != domain.AccessReasonLocked {
		t.Fatalf("%+v", items)
	}
}

func TestBookService_RecentlyAdded_Top5NewestFirst(t *testing.T) {
	br, er := newFakeCatalog()
	base := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := range 7 {
		id := uuid.New()
		at := base.Add(time.Duration(i) * time.Hour)
		b := &domain.Book{
			ID: id, Title: "B", Language: "en", PriceCents: int32(i),
			AddedAt: at, Content: "body",
		}
		br.byID[id] = b
		br.list = append(br.list, *b)
	}
	// Shuffle insertion order in list to ensure service/repo sort by added_at, not slice order.
	br.list[0], br.list[1], br.list[2], br.list[3], br.list[4], br.list[5], br.list[6] =
		br.list[3], br.list[6], br.list[1], br.list[4], br.list[0], br.list[2], br.list[5]

	svc := NewBookService(br, er)
	items, err := svc.RecentlyAdded(context.Background(), uuid.Nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 5 {
		t.Fatalf("want 5 items, got %d", len(items))
	}
	for i := range items {
		if items[i].Content != "" {
			t.Fatalf("item %d: content should be empty", i)
		}
		if items[i].IsAccessible || items[i].AccessReason != domain.AccessReasonLocked {
			t.Fatalf("guest item %d: want locked", i)
		}
	}
	// Newest five are added_at base+6h..+2h → price_cents 6,5,4,3,2
	want := []int32{6, 5, 4, 3, 2}
	for i, w := range want {
		if items[i].PriceCents != w {
			t.Fatalf("idx %d: price_cents=%d want %d (order newest first)", i, items[i].PriceCents, w)
		}
	}
}

func TestBookService_RecentlyAdded_SubscriptionMarksAccessible(t *testing.T) {
	br, er := newFakeCatalog()
	u := uuid.New()
	oldT := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newT := oldT.Add(24 * time.Hour)
	oldID, newID := uuid.New(), uuid.New()
	br.byID[oldID] = &domain.Book{ID: oldID, Title: "Old", Language: "en", PriceCents: 1, AddedAt: oldT}
	br.byID[newID] = &domain.Book{ID: newID, Title: "New", Language: "en", PriceCents: 2, AddedAt: newT}
	br.list = []domain.Book{*br.byID[oldID], *br.byID[newID]}
	now := time.Now()
	end := now.AddDate(0, 0, domain.SubscriptionPeriodDays)
	_, _ = er.Create(context.Background(), u, nil, domain.EntitlementSubscription, domain.EntitlementActive, &end, &now)

	svc := NewBookService(br, er)
	items, err := svc.RecentlyAdded(context.Background(), u, domain.RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d", len(items))
	}
	if items[0].ID != newID || !items[0].IsAccessible {
		t.Fatalf("first should be newest and accessible: %+v", items[0])
	}
	if !items[1].IsAccessible {
		t.Fatalf("second should be accessible with sub: %+v", items[1])
	}
}

func TestBookService_MyLibrary(t *testing.T) {
	br, er := newFakeCatalog()
	u := uuid.New()
	bid := uuid.New()
	br.byID[bid] = &domain.Book{ID: bid, Title: "Owned", Content: "secret", Language: "en", PriceCents: 9, AddedAt: time.Now()}
	br.list = []domain.Book{*br.byID[bid]}
	er.books[bid] = true
	_, _ = er.Create(context.Background(), u, &bid, domain.EntitlementSinglePurchase, domain.EntitlementActive, nil, nil)

	svc := NewBookService(br, er)
	lib, err := svc.MyLibrary(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if lib.Subscription != nil {
		t.Fatal("expected no subscription")
	}
	if len(lib.Purchases) != 1 || lib.Purchases[0].Book.Title != "Owned" || lib.Purchases[0].Book.Content != "" {
		t.Fatalf("%+v", lib)
	}
}
