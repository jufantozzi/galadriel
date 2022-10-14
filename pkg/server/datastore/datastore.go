package datastore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/HewlettPackard/galadriel/pkg/common"
	"github.com/google/uuid"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

type DataStore interface {
	CreateMember(ctx context.Context, m *common.Member) (*Member, error)
	UpdateMember(ctx context.Context, trustDomain string, m *common.Member) (*common.Member, error)
	GetMember(ctx context.Context, trustDomain string) (*common.Member, error)
	ListMembers(ctx context.Context) ([]*common.Member, error)

	CreateRelationship(ctx context.Context, r *common.Relationship) (*common.Relationship, error)
	GetRelationships(ctx context.Context, trustDomain string) ([]*common.Relationship, error)

	GenerateAccessToken(ctx context.Context, t *common.AccessToken, trustDomain string) (*common.AccessToken, error)
	GetAccessToken(ctx context.Context, token string) (*common.AccessToken, error)
	ListRelationships(ctx context.Context) ([]*common.Relationship, error)
}

type AccessToken struct {
	Token  string
	Expiry time.Time
	Member *Member
}

type Member struct {
	ID uuid.UUID

	Name         string
	TrustDomain  string
	TrustBundle  []byte
	BundleDigest []byte
}

type Relationship struct {
	ID uuid.UUID

	MemberA *Member
	MemberB *Member
}

// TODO: use until an actual DataStore implementation is added.

type MemStore struct {
	members      map[string]*Member // trust_domain (e.g. 'example.org') -> member
	relationship []*Relationship
	tokens       map[string]*AccessToken // token uuid string -> access token

	mu sync.RWMutex
}

func NewMemStore() DataStore {
	return &MemStore{
		members: make(map[string]*Member),
		tokens:  make(map[string]*AccessToken),
		mu:      sync.RWMutex{},
	}
}

func (s *MemStore) CreateMember(_ context.Context, member *common.Member) (*Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exist := s.members[member.TrustDomain.String()]; exist {
		return nil, fmt.Errorf("member already exists: %s", member.TrustDomain)
	}

	m := &Member{
		ID:          uuid.New(),
		Name:        member.Name,
		TrustDomain: member.TrustDomain.String(),
	}

	s.members[m.TrustDomain] = m

	return m, nil
}

func (s *MemStore) UpdateMember(_ context.Context, trustDomain string, member *common.Member) (*common.Member, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if member == nil {
		return nil, errors.New("failed to updated member: no member data given")
	}

	m, ok := s.members[trustDomain]
	if !ok {
		return nil, errors.New("failed updating member: trust domain not found: " + trustDomain)
	}

	if len(member.Bundle) != 0 {
		m.TrustBundle = member.Bundle
	}

	if len(member.BundleDigest) != 0 {
		m.BundleDigest = member.BundleDigest
	}

	return &common.Member{
		ID:          member.ID,
		Name:        member.Name,
		TrustBundle: member.TrustBundle,
	}, nil
}

func (s *MemStore) GetMember(_ context.Context, trustDomain string) (*common.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.members[trustDomain]
	if !ok {
		return nil, errors.New("failed getting member: trust domain not found: " + trustDomain)
	}

	td, err := spiffeid.TrustDomainFromString(m.TrustDomain)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain: %v", err)
	}

	return &common.Member{
		ID:   m.ID,
		Name: m.Name,
		TrustBundle: common.TrustBundle{
			TrustDomain:  td,
			Bundle:       m.TrustBundle,
			BundleDigest: m.BundleDigest,
		},
	}, nil
}

func (s *MemStore) ListMembers(ctx context.Context) ([]*common.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var members []*common.Member
	for _, m := range s.members {
		td, err := spiffeid.TrustDomainFromString(m.TrustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain: %v", err)
		}

		members = append(members, &common.Member{
			ID:          m.ID,
			Name:        m.Name,
			TrustBundle: common.TrustBundle{TrustDomain: td},
		})
	}

	return members, nil
}

func (s *MemStore) CreateRelationship(_ context.Context, rel *common.Relationship) (*common.Relationship, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rel.MemberA.TrustDomain.Compare(rel.MemberB.TrustDomain) == 0 {
		return nil, fmt.Errorf("cannot create relationship: trust domain members are the same: %s", rel.MemberA.TrustDomain)
	}

	if _, ok := s.members[rel.MemberA.TrustDomain.String()]; !ok {
		return nil, fmt.Errorf("member not found for trust domain: %s", rel.MemberA.TrustDomain)
	}
	if _, ok := s.members[rel.MemberB.TrustDomain.String()]; !ok {
		return nil, fmt.Errorf("member not found for trust domain: %s", rel.MemberB.TrustDomain)
	}
	r := &Relationship{
		ID:      uuid.New(),
		MemberA: s.members[rel.MemberA.TrustDomain.String()],
		MemberB: s.members[rel.MemberB.TrustDomain.String()],
	}

	s.relationship = append(s.relationship, r)

	rel.ID = r.ID
	return rel, nil
}

func (s *MemStore) GetRelationships(_ context.Context, trustDomain string) ([]*common.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var response []*common.Relationship

	for _, r := range s.relationship {
		if r.MemberA.TrustDomain != trustDomain && r.MemberB.TrustDomain != trustDomain {
			continue
		}
		_, ok := s.members[r.MemberA.TrustDomain]
		if !ok {
			return nil, errors.New("failed getting relationship: memberA not found")
		}
		_, ok = s.members[r.MemberB.TrustDomain]
		if !ok {
			return nil, errors.New("failed getting relationship: memberB not found")
		}
		tdA, err := spiffeid.TrustDomainFromString(r.MemberA.TrustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain: %v", err)
		}
		tdB, err := spiffeid.TrustDomainFromString(r.MemberB.TrustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain: %v", err)
		}

		response = append(response, &common.Relationship{
			ID: r.ID,
			MemberA: &common.Member{
				ID:   r.MemberA.ID,
				Name: r.MemberA.Name,
				TrustBundle: common.TrustBundle{
					TrustDomain:  tdA,
					Bundle:       r.MemberA.TrustBundle,
					BundleDigest: r.MemberA.BundleDigest,
				},
			},
			MemberB: &common.Member{
				ID:   r.MemberB.ID,
				Name: r.MemberB.Name,
				TrustBundle: common.TrustBundle{
					TrustDomain:  tdB,
					Bundle:       r.MemberB.TrustBundle,
					BundleDigest: r.MemberB.BundleDigest,
				},
			},
		})
	}

	return response, nil
}

func (s *MemStore) ListRelationships(ctx context.Context) ([]*common.Relationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rels []*common.Relationship
	for _, r := range s.relationship {
		tdA, err := spiffeid.TrustDomainFromString(r.MemberA.TrustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain: %v", err)
		}
		tdB, err := spiffeid.TrustDomainFromString(r.MemberB.TrustDomain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain: %v", err)
		}
		rels = append(rels, &common.Relationship{
			ID: r.ID,
			MemberA: &common.Member{
				ID:   r.MemberA.ID,
				Name: r.MemberA.Name,
				TrustBundle: common.TrustBundle{
					TrustDomain:  tdA,
					Bundle:       r.MemberA.TrustBundle,
					BundleDigest: r.MemberA.BundleDigest,
				},
			},
			MemberB: &common.Member{
				ID:   r.MemberB.ID,
				Name: r.MemberB.Name,
				TrustBundle: common.TrustBundle{
					TrustDomain:  tdB,
					Bundle:       r.MemberB.TrustBundle,
					BundleDigest: r.MemberB.BundleDigest,
				},
			},
		})
	}

	return rels, nil
}

func (s *MemStore) GenerateAccessToken(_ context.Context, token *common.AccessToken, trustDomain string) (*common.AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	member := s.members[trustDomain]
	if member == nil {
		return nil, fmt.Errorf("failed to find member for the trust domain: %s", trustDomain)
	}

	at := &AccessToken{
		Token:  token.Token,
		Expiry: token.Expiry,
		Member: member,
	}

	s.tokens[at.Token] = at

	token.MemberID = member.ID
	return token, nil
}

func (s *MemStore) GetAccessToken(_ context.Context, token string) (*common.AccessToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	at, ok := s.tokens[token]
	if !ok {
		return nil, errors.New("failed to find token")
	}

	td, err := spiffeid.TrustDomainFromString(at.Member.TrustDomain)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain: %v", err)
	}
	return &common.AccessToken{
		MemberID:    at.Member.ID,
		TrustDomain: td,
		Token:       at.Token,
		Expiry:      at.Expiry,
	}, nil
}
