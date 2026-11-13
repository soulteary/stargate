package handlers

import (
	"encoding/xml"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

type xmlErrorDocument struct {
	XMLName xml.Name `xml:"errors"`
	Errors  []struct {
		Code    int    `xml:"code,attr"`
		Message string `xml:",chardata"`
		Nested  []struct {
			XMLName xml.Name
		} `xml:",any"`
	} `xml:"error"`
}

func TestSendErrorResponseEscapesXMLMessage(t *testing.T) {
	for _, message := range []string{
		`</error><error code="200">ok`,
		`a < b && c > "d" 'e'`,
		"<!-- comment --><![CDATA[x]]>",
		"plain message",
	} {
		t.Run(message, func(t *testing.T) {
			ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{"Accept": "application/xml"}, "")
			defer app.ReleaseCtx(ctx)

			require.NoError(t, SendErrorResponse(ctx, fiber.StatusUnauthorized, message))
			require.Equal(t, fiber.StatusUnauthorized, ctx.Response().StatusCode())
			require.Equal(t, "application/xml", string(ctx.Response().Header.ContentType()))

			var document xmlErrorDocument
			require.NoError(t, xml.Unmarshal(ctx.Response().Body(), &document), string(ctx.Response().Body()))
			require.Len(t, document.Errors, 1)
			require.Equal(t, fiber.StatusUnauthorized, document.Errors[0].Code)
			require.Equal(t, message, document.Errors[0].Message)
			require.Empty(t, document.Errors[0].Nested)
		})
	}
}

func TestSendErrorResponseReplacesCharactersXMLDoesNotAllow(t *testing.T) {
	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{"Accept": "application/xml"}, "")
	defer app.ReleaseCtx(ctx)

	require.NoError(t, SendErrorResponse(ctx, fiber.StatusBadRequest, "a\x00b\x1bc\uFFFEd\xffe\tf\ng"))

	var document xmlErrorDocument
	require.NoError(t, xml.Unmarshal(ctx.Response().Body(), &document), string(ctx.Response().Body()))
	require.Len(t, document.Errors, 1)
	require.Equal(t, "a\uFFFDb\uFFFDc\uFFFDd\uFFFDe\tf\ng", document.Errors[0].Message)
}

// Messages without markup, including quotes and non-ASCII text, keep the exact
// bytes they had before.
func TestSendErrorResponseKeepsQuotedXMLBody(t *testing.T) {
	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{"Accept": "application/xml"}, "")
	defer app.ReleaseCtx(ctx)

	require.NoError(t, SendErrorResponse(ctx, fiber.StatusUnauthorized, `l'utilisateur "x" n'est pas autorisé：验证码`))
	require.Equal(t, `<errors><error code="401">l'utilisateur "x" n'est pas autorisé：验证码</error></errors>`, string(ctx.Response().Body()))
}

// Messages without markup keep the exact bytes they had before.
func TestSendErrorResponseKeepsPlainXMLBody(t *testing.T) {
	ctx, app := createTestContext(fiber.MethodGet, "/_auth", map[string]string{"Accept": "application/xml"}, "")
	defer app.ReleaseCtx(ctx)

	require.NoError(t, SendErrorResponse(ctx, fiber.StatusForbidden, "cross-site request rejected"))
	require.Equal(t, `<errors><error code="403">cross-site request rejected</error></errors>`, string(ctx.Response().Body()))
}
