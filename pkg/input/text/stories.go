package text

import (
	"context"

	"github.com/cfoust/cy/pkg/geom"
	"github.com/cfoust/cy/pkg/mux"
	"github.com/cfoust/cy/pkg/stories"
)

var BottomRight stories.InitFunc = func(ctx context.Context) (mux.Screen, error) {
	location := geom.DEFAULT_SIZE
	location.C -= 1
	location.R -= 1
	return New(
		ctx,
		WithInline(location, geom.DEFAULT_SIZE),
	), nil
}

var TopLeft stories.InitFunc = func(ctx context.Context) (mux.Screen, error) {
	return New(
		ctx,
		WithInline(geom.Size{}, geom.DEFAULT_SIZE),
	), nil
}

var FullTop stories.InitFunc = func(ctx context.Context) (mux.Screen, error) {
	f := New(
		ctx,
	)
	return f, nil
}

var FullBottom stories.InitFunc = func(ctx context.Context) (mux.Screen, error) {
	f := New(
		ctx,
	)
	return f, nil
}

var Placeholder stories.InitFunc = func(ctx context.Context) (mux.Screen, error) {
	f := New(
		ctx,
		WithPlaceholder("placeholder"),
		WithPrompt("placeholder test"),
	)
	return f, nil
}

var Preset stories.InitFunc = func(ctx context.Context) (mux.Screen, error) {
	f := New(
		ctx,
		WithPreset("preset text"),
	)
	return f, nil
}

func init() {
	config := stories.Config{
		Size: geom.DEFAULT_SIZE,
		Input: []interface{}{
			stories.Wait(stories.Some),
			stories.Type("this is a test"),
			stories.Wait(stories.ALot),
		},
	}
	stories.Register("input/text/bottom-right", BottomRight, config)
	stories.Register("input/text/top-left", TopLeft, config)
	stories.Register("input/text/full-top", FullTop, config)
	stories.Register("input/text/full-bottom", FullBottom, config)
	stories.Register("input/text/placeholder", Placeholder, config)
	stories.Register("input/text/preset", Preset, config)
}
