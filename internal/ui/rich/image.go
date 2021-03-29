package rich

import "log"

// ImageURLSetter describes an image that can be set a URL.
type ImageURLSetter interface {
	SetImageURL(url string)
}

type tooltipSetter interface {
	SetTooltipText(text string)
}

// BindRoundImage binds a round image to a rich label state store.
func BindRoundImage(img ImageURLSetter, state LabelStateStorer, tooltip bool) {
	var setTooltip func(string)

	if tooltip {
		tooltipper, ok := img.(tooltipSetter)
		if !ok {
			log.Panicf("img of type %T is not tooltipSetter", img)
		}

		setTooltip = tooltipper.SetTooltipText
	}

	state.OnUpdate(func() {
		image := state.Image()
		img.SetImageURL(image.URL)

		if setTooltip != nil {
			setTooltip(state.Label().String())
		}
	})
}
