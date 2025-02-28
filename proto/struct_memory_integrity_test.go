package proto

type ContactTelephoneKind int

const (
	MOBILE ContactTelephoneKind = 0
	WORK   ContactTelephoneKind = 1
	HOME   ContactTelephoneKind = 2
	HOUSE  ContactTelephoneKind = 2
)

type ContactTelephone struct {
	Kind   ContactTelephoneKind `arf:"0"`
	Number string               `arf:"1"`
}

func (ContactTelephone) ArfStructID() string { return "org.example.contacts/ContactTelephone" }

// Contact represent a single person in the address list.
type Contact struct {
	Id             *int64             `arf:"0"`
	Name           string             `arf:"1"`
	Surname        string             `arf:"2"`
	Company        *Company           `arf:"3"`
	Emails         []string           `arf:"4"`
	Telephones     []ContactTelephone `arf:"5"`
	AdditionalInfo map[string]string  `arf:"6"`
}

func (Contact) ArfStructID() string { return "org.example.contacts/Contact" }

// Company represents a company in which a person
// works at.
type Company struct {
	Name           string `arf:"0"`
	WebsiteAddress string `arf:"1"`
}

func (Company) ArfStructID() string { return "org.example.contacts/Company" }

// GetContactRequest represents a request to obtain
// a specific contact through a given id.
type GetContactRequest struct {
	Id int64 `arf:"0"`
}

func (GetContactRequest) ArfStructID() string { return "org.example.contacts/GetContactRequest" }

// GetContactResponse represents the result of a GetContactRequest.
// An absent `contact` indicates that no contact under the provided id exists.
type GetContactResponse struct {
	Contact *Contact `arf:"0"`
}

func (GetContactResponse) ArfStructID() string { return "org.example.contacts/GetContactResponse" }

//func TestStructMemoryIntegrity(t *testing.T) {
//	resetRegistry()
//	RegisterMessage(ContactTelephone{})
//	RegisterMessage(Contact{})
//	RegisterMessage(Company{})
//	RegisterMessage(GetContactRequest{})
//	RegisterMessage(GetContactResponse{})
//
//	d, err := hex.DecodeString(strings.ReplaceAll("08041C6F 72672E65 78616D70 6C652E63 6F6E7461 6374732F 436F6E74 616374C3 01000001 04044665 666F0204 084D6172 696F7474 69030804 1C6F7267 2E657861 6D706C65 2E636F6E 74616374 732F436F 6D70616E 792A0004 104D6172 696F7474 6920436F 6D70616E 79010414 68747470 733A2F2F 6D617269 6F747469 2E646576 04060104 0D666566 6F406665 666F2E64 65630506 01080425 6F72672E 6578616D 706C652E 636F6E74 61637473 2F436F6E 74616374 54656C65 70686F6E 650A0031 01040531 32333435 06090704 08666566 6F666566 6F09070C 01040474 65737404 03796570", " ", ""))
//	require.NoError(t, err)
//	v, err := DecodeAny(bytes.NewReader(d))
//	require.NoError(t, err)
//	c := v.(*Contact)
//
//	assert.Equal(t, "Fefo", c.Name)
//	assert.Equal(t, "Mariotti", c.Surname)
//	assert.Equal(t, "Mariotti Company", c.Company.Name)
//	assert.Equal(t, "https://mariotti.dev", c.Company.WebsiteAddress)
//	assert.Equal(t, []string{"fefo@fefo.dec"}, c.Emails)
//	assert.Equal(t, MOBILE, c.Telephones[0].Kind)
//	assert.Equal(t, "12345", c.Telephones[0].Number)
//	assert.Equal(t, map[string]string{"test": "yep"}, c.AdditionalInfo)
//}
