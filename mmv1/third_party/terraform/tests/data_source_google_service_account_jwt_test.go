package google

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	jwtTestSubject          = "custom-subject"
	jwtTestFoo              = "bar"
	jwtTestComplexFooNested = "baz"
)

type jwtTestPayload struct {
	Subject string `json:"sub"`

	Foo string `json:"foo"`

	ComplexFoo struct {
		Nested string `json:"nested"`
	} `json:"complexFoo"`
}

func testAccCheckServiceAccountJwtValue(name, audience string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()

		rs, ok := ms.Resources[name]

		if !ok {
			return fmt.Errorf("can't find %s in state", name)
		}

		jwtString, ok := rs.Primary.Attributes["jwt"]

		if !ok {
			return fmt.Errorf("jwt not found")
		}

		jwtParts := strings.Split(jwtString, ".")

		if len(jwtParts) != 3 {
			return errors.New("jwt does not appear well-formed")
		}

		decoded, err := base64.RawURLEncoding.DecodeString(jwtParts[1])

		if err != nil {
			return fmt.Errorf("could not base64 decode jwt body: %w", err)
		}

		var payload jwtTestPayload

		err = json.NewDecoder(bytes.NewBuffer(decoded)).Decode(&payload)

		if err != nil {
			return fmt.Errorf("could not decode jwt payload: %w", err)
		}

		if payload.Subject != jwtTestSubject {
			return fmt.Errorf("invalid 'sub', expected '%s', got '%s'", jwtTestSubject, payload.Subject)
		}

		if payload.Foo != jwtTestFoo {
			return fmt.Errorf("invalid 'foo', expected '%s', got '%s'", jwtTestFoo, payload.Foo)
		}

		if payload.ComplexFoo.Nested != jwtTestComplexFooNested {
			return fmt.Errorf("invalid 'foo', expected '%s', got '%s'", jwtTestComplexFooNested, payload.ComplexFoo.Nested)
		}

		return nil
	}
}

func TestAccDataSourceGoogleServiceAccountJwt(t *testing.T) {
	t.Parallel()

	resourceName := "data.google_service_account_jwt.default"
	serviceAccount := getTestServiceAccountFromEnv(t)
	targetServiceAccountEmail := BootstrapServiceAccount(t, getTestProjectFromEnv(), serviceAccount)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGoogleServiceAccountJwt(targetServiceAccountEmail),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountJwtValue(resourceName, targetAudience),
				),
			},
		},
	})
}

func testAccCheckGoogleServiceAccountJwt(targetServiceAccount string) string {
	return fmt.Sprintf(`
data "google_service_account_jwt" "default" {
	target_service_account = "%s"

    payload = jsonencode({
      sub: "%s",
      foo: "%s",
      complexFoo: {
        nested: "%s"
      }
    })
}
`, targetServiceAccount, jwtTestSubject, jwtTestFoo, jwtTestComplexFooNested)
}
