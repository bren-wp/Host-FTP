package desktop

import "testing"

func TestStatusBandGeometryKeepsFooterInsideMinimumWorkspace(t *testing.T) {
	statusY, contentBottom := statusBandGeometry(premiumMinHeight)

	wantStatusY := premiumMinHeight - statusBandHeight - statusBandBottomInset
	wantContentBottom := wantStatusY - statusBandContentGap
	if statusY != wantStatusY {
		t.Fatalf("minimum-height status y = %d, want %d", statusY, wantStatusY)
	}
	if contentBottom != wantContentBottom {
		t.Fatalf("minimum-height content bottom = %d, want %d", contentBottom, wantContentBottom)
	}
	if got := premiumMinHeight - (statusY + statusBandHeight); got != statusBandBottomInset {
		t.Fatalf("status bottom inset = %d, want %d", got, statusBandBottomInset)
	}
	if got := statusY - contentBottom; got != statusBandContentGap {
		t.Fatalf("content/status gap = %d, want %d", got, statusBandContentGap)
	}
}

func TestStatusBandGeometryUsesActualConstrainedClientHeight(t *testing.T) {
	const clientHeight = 640
	statusY, contentBottom := statusBandGeometry(clientHeight)

	if statusY != 581 || contentBottom != 574 {
		t.Fatalf("constrained geometry = (%d, %d), want (581, 574)", statusY, contentBottom)
	}
	if statusY+statusBandHeight+statusBandBottomInset != clientHeight {
		t.Fatalf("footer does not terminate inside actual client height %d", clientHeight)
	}
}

func TestStatusBandGeometryClampsTinyHeight(t *testing.T) {
	statusY, contentBottom := statusBandGeometry(1)
	if statusY != 0 || contentBottom != 0 {
		t.Fatalf("tiny-height geometry = (%d, %d), want (0, 0)", statusY, contentBottom)
	}
}
