// Copyright 2010 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package walk

import (
	"math"
	"strconv"

	"github.com/lxn/win"
)

type ImageViewMode int

const (
	ImageViewModeIdeal ImageViewMode = iota
	ImageViewModeCorner
	ImageViewModeCenter
	ImageViewModeShrink
	ImageViewModeZoom
	ImageViewModeStretch
)

type ImageView struct {
	*CustomWidget
	image                  Image
	imageChangedPublisher  EventPublisher
	margin96dpi            int
	marginChangedPublisher EventPublisher
	mode                   ImageViewMode
}

func NewImageView(parent Container) (*ImageView, error) {
	iv := new(ImageView)

	cw, err := NewCustomWidget(parent, 0, func(canvas *Canvas, updateBounds Rectangle) error {
		return iv.drawImage(canvas, updateBounds)
	})
	if err != nil {
		return nil, err
	}

	iv.CustomWidget = cw

	if err := InitWrapperWindow(iv); err != nil {
		iv.Dispose()
		return nil, err
	}

	iv.SetInvalidatesOnResize(true)

	iv.SetBackground(NullBrush())

	iv.MustRegisterProperty("Image", NewProperty(
		func() interface{} {
			return iv.Image()
		},
		func(v interface{}) error {
			var img Image

			switch val := v.(type) {
			case Image:
				img = val

			case int:
				var err error
				if img, err = Resources.Image(strconv.Itoa(val)); err != nil {
					return err
				}

			case string:
				var err error
				if img, err = Resources.Image(val); err != nil {
					return err
				}

			default:
				return ErrInvalidType
			}

			return iv.SetImage(img)
		},
		iv.imageChangedPublisher.Event()))

	iv.MustRegisterProperty("Margin", NewProperty(
		func() interface{} {
			return iv.Margin()
		},
		func(v interface{}) error {
			return iv.SetMargin(assertIntOr(v, 0))
		},
		iv.MarginChanged()))

	return iv, nil
}

func (iv *ImageView) Mode() ImageViewMode {
	return iv.mode
}

func (iv *ImageView) SetMode(mode ImageViewMode) {
	if mode == iv.mode {
		return
	}

	iv.mode = mode

	iv.Invalidate()

	iv.RequestLayout()
}

func (iv *ImageView) applyDPI(dpi int) {
	iv.CustomWidget.ApplyDPI(dpi)

	iv.Invalidate()

	iv.RequestLayout()
}

func (iv *ImageView) Image() Image {
	return iv.image
}

func (iv *ImageView) SetImage(image Image) error {
	if image == iv.image {
		return nil
	}

	var oldSize, newSize Size
	if iv.image != nil {
		oldSize = iv.image.Size()
	}
	if image != nil {
		newSize = image.Size()
	}

	iv.image = image

	_, isMetafile := image.(*Metafile)
	iv.SetClearsBackground(isMetafile)

	err := iv.Invalidate()

	if iv.mode == ImageViewModeIdeal && newSize != oldSize {
		iv.RequestLayout()
	}

	iv.imageChangedPublisher.Publish()

	return err
}

func (iv *ImageView) ImageChanged() *Event {
	return iv.imageChangedPublisher.Event()
}

func (iv *ImageView) Margin() int {
	return iv.margin96dpi
}

func (iv *ImageView) SetMargin(margin int) error {
	if margin == iv.margin96dpi {
		return nil
	}

	iv.margin96dpi = margin

	err := iv.Invalidate()

	if iv.mode == ImageViewModeIdeal {
		iv.RequestLayout()
	}

	iv.marginChangedPublisher.Publish()

	return err
}

func (iv *ImageView) MarginChanged() *Event {
	return iv.marginChangedPublisher.Event()
}

func (iv *ImageView) drawImage(canvas *Canvas, _ Rectangle) error {
	if iv.image == nil {
		return nil
	}

	cb := iv.ClientBoundsPixels()

	margin := iv.IntFrom96DPI(iv.margin96dpi)

	cb.Width -= margin * 2
	cb.Height -= margin * 2

	s := iv.SizeFrom96DPI(iv.image.Size())

	switch iv.mode {
	case ImageViewModeShrink, ImageViewModeZoom, ImageViewModeStretch:
		var bounds Rectangle

		if iv.mode == ImageViewModeStretch {
			bounds.X = margin
			bounds.Y = margin
			bounds.Width = cb.Width
			bounds.Height = cb.Height
		} else {
			var scale float64
			if iv.mode == ImageViewModeZoom || s.Width > cb.Width || s.Height > cb.Height {
				sx := float64(cb.Width) / float64(s.Width)
				sy := float64(cb.Height) / float64(s.Height)

				scale = math.Min(sx, sy)
			} else {
				scale = 1.0
			}

			s = scaleSize(s, scale)

			bounds.Width = s.Width
			bounds.Height = s.Height
			bounds.X = margin + (cb.Width-bounds.Width)/2
			bounds.Y = margin + (cb.Height-bounds.Height)/2
		}

		return canvas.DrawImageStretched(iv.image, bounds.To96DPI(iv.DPI()))

	case ImageViewModeCorner, ImageViewModeCenter:
		win.IntersectClipRect(canvas.hdc, int32(margin), int32(margin), int32(cb.Width+margin), int32(cb.Height+margin))
	}

	var pos Point

	switch iv.mode {
	case ImageViewModeIdeal, ImageViewModeCorner:
		pos.X = margin
		pos.Y = margin

	case ImageViewModeCenter:
		pos.X = margin + (cb.Width-s.Width)/2
		pos.Y = margin + (cb.Height-s.Height)/2
	}

	return canvas.DrawImage(iv.image, pos.To96DPI(iv.DPI()))
}

func (iv *ImageView) CreateLayoutItem(ctx *LayoutContext) LayoutItem {
	var layoutFlags LayoutFlags
	if iv.mode != ImageViewModeIdeal {
		layoutFlags = ShrinkableHorz | ShrinkableVert | GrowableHorz | GrowableVert | GreedyHorz | GreedyVert
	}

	idealSize := iv.SizeFrom96DPI(Size{100, 100})

	var minSize Size
	if iv.mode == ImageViewModeIdeal {
		if iv.image != nil {
			m2 := iv.IntFrom96DPI(iv.margin96dpi) * 2
			// TODO: If image is Bitmap, Size() returns pixels. If image is Icon, Size() returns 96dpi pixels.
			s := iv.SizeFrom96DPI(iv.image.Size())
			s.Width += m2
			s.Height += m2
			idealSize = s
		}

		minSize = idealSize
	} else {
		s := iv.IntFrom96DPI(iv.margin96dpi)*2 + 1
		minSize = Size{s, s}
	}

	return &imageViewLayoutItem{
		layoutFlags: layoutFlags,
		idealSize:   idealSize,
		minSize:     minSize,
	}
}

type imageViewLayoutItem struct {
	LayoutItemBase
	layoutFlags LayoutFlags
	idealSize   Size
	minSize     Size
}

func (li *imageViewLayoutItem) LayoutFlags() LayoutFlags {
	return li.layoutFlags
}

func (li *imageViewLayoutItem) IdealSize() Size {
	return li.idealSize
}

func (li *imageViewLayoutItem) MinSize() Size {
	return li.minSize
}
