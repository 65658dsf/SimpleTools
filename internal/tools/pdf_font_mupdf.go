//go:build cgo && mupdf && !nocgo

package tools

/*
MuPDF's prebuilt go-fitz archives intentionally omit their bundled CJK fonts.
Keep the fallback font in the application and install it through MuPDF's CJK
and generic fallback callbacks for every document context.

The declarations below intentionally mirror the small public subset of the
MuPDF API we use. go-fitz owns the platform-specific include and linker flags,
so this package does not duplicate or hard-code its module-cache paths.
*/
/*
#include <stddef.h>
#include <stdlib.h>
#include <string.h>

typedef struct fz_context fz_context;
typedef struct fz_font fz_font;

typedef fz_font *(st_load_system_font_fn)(fz_context *ctx, const char *name,
    int bold, int italic, int needs_exact_metrics);
typedef fz_font *(st_load_system_cjk_font_fn)(fz_context *ctx, const char *name,
    int ordering, int serif);
typedef fz_font *(st_load_system_fallback_font_fn)(fz_context *ctx, int script,
    int language, int serif, int bold, int italic);

extern void fz_install_load_system_font_funcs(fz_context *ctx,
    st_load_system_font_fn *f, st_load_system_cjk_font_fn *f_cjk,
    st_load_system_fallback_font_fn *f_fallback);
extern fz_font *fz_new_font_from_memory(fz_context *ctx, const char *name,
    const unsigned char *data, int len, int index, int use_glyph_bbox);

static const unsigned char *st_cjk_font_data;
static int st_cjk_font_len;

static fz_font *st_load_cjk_font(fz_context *ctx, const char *name,
    int ordering, int serif) {
    (void)name;
    (void)ordering;
    (void)serif;
    if (st_cjk_font_data == NULL || st_cjk_font_len <= 0) {
        return NULL;
    }
    // The embedded asset is validated at build time; MuPDF owns the returned
    // font and creates a fresh handle for each callback invocation.
    return fz_new_font_from_memory(ctx, "SimpleTools Noto Sans CJK SC",
        st_cjk_font_data, st_cjk_font_len, 0, 0);
}

static fz_font *st_load_fallback_font(fz_context *ctx, int script,
    int language, int serif, int bold, int italic) {
    (void)script;
    (void)language;
    (void)serif;
    (void)bold;
    (void)italic;
    if (st_cjk_font_data == NULL || st_cjk_font_len <= 0) {
        return NULL;
    }
    return fz_new_font_from_memory(ctx, "SimpleTools Noto Sans CJK SC",
        st_cjk_font_data, st_cjk_font_len, 0, 0);
}

static int st_prepare_cjk_font(const unsigned char *data, size_t len) {
    unsigned char *copy;
    if (data == NULL || len == 0 || len > 2147483647U) {
        return 0;
    }
    if (st_cjk_font_data != NULL) {
        return st_cjk_font_len == (int)len;
    }
    copy = (unsigned char *)malloc(len);
    if (copy == NULL) {
        return 0;
    }
    memcpy(copy, data, len);
    st_cjk_font_data = copy;
    st_cjk_font_len = (int)len;
    return 1;
}

static void st_install_cjk_font_funcs(fz_context *ctx) {
    fz_install_load_system_font_funcs(ctx, NULL, st_load_cjk_font,
        st_load_fallback_font);
}
*/
import "C"

import (
	_ "embed"
	"errors"
	"reflect"
	"sync"
	"unsafe"

	fitz "github.com/gen2brain/go-fitz"
)

//go:embed assets/NotoSansSC-Regular.ttf
var cjkFontData []byte

var (
	cjkFontOnce sync.Once
	cjkFontOK   bool
)

// installCJKFallback installs the callback on the private MuPDF context held
// by go-fitz. The context is deliberately obtained by field name so a future
// go-fitz layout change fails closed instead of dereferencing an arbitrary
// pointer.
func installCJKFallback(doc *fitz.Document) error {
	if doc == nil {
		return errors.New("cannot install CJK fallback on a nil PDF document")
	}
	value := reflect.ValueOf(doc)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return errors.New("invalid PDF document value")
	}
	field := value.Elem().FieldByName("ctx")
	if !field.IsValid() || field.Kind() != reflect.Pointer || field.IsNil() {
		return errors.New("go-fitz document context is unavailable")
	}
	ctx := unsafe.Pointer(field.Pointer())
	if ctx == nil {
		return errors.New("go-fitz document context is nil")
	}

	cjkFontOnce.Do(func() {
		if len(cjkFontData) == 0 {
			return
		}
		cjkFontOK = C.st_prepare_cjk_font(
			(*C.uchar)(unsafe.Pointer(&cjkFontData[0])), C.size_t(len(cjkFontData))) != 0
	})
	if !cjkFontOK {
		return errors.New("cannot prepare embedded CJK fallback font")
	}

	C.st_install_cjk_font_funcs((*C.struct_fz_context)(ctx))
	return nil
}
