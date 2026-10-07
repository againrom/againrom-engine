package audio

import "image"

func FixedRequest(source, selector string, group Channel, priority uint8, repeat bool, placement Placement) Request {
	return Request{Source: source, Recipe: source, VolumeTerm: "setting", Selector: selector, Group: group, Priority: priority,
		Repeat: repeat, Placement: placement, SpatialKnown: true}
}

func PositionalRequest(source, selector string, group Channel, fine image.Point, geometry ViewGeometry) (Request, bool) {
	terms, ok := PositionalTerms(fine, geometry)
	if !ok {
		return Request{}, false
	}
	request := FixedRequest(source, selector, group, uint8((10000-abs(terms.Attenuation))/100), false, SpatialPlacement(terms))
	request.VolumeTerm, request.Attenuation, request.Pan = "distance+setting", terms.Attenuation, terms.Pan
	request.Geometry, request.SourceFine = geometry, fine
	return request, true
}

func AmbientRequest(source, selector string, repeat bool, cells []image.Point, geometry ViewGeometry, populationGain bool) (Request, bool) {
	terms, ok := AmbientTerms(cells, geometry, populationGain)
	if !ok {
		return Request{}, false
	}
	request := FixedRequest(source, selector, EffectsChannel, 220, repeat, SpatialPlacement(terms))
	request.VolumeTerm, request.Attenuation, request.Pan = "ambient+setting", terms.Attenuation, terms.Pan
	request.Geometry = geometry
	return request, true
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

type SampleRequester interface {
	RequestSample(Sample, Request) Voice
}

func Dispatch(player Player, sample Sample, request Request) Voice {
	if typed, ok := player.(SampleRequester); ok {
		return typed.RequestSample(sample, request)
	}
	return nil
}

func StopReset(voice Voice) {
	if reset, ok := voice.(interface{ StopReset() }); ok {
		reset.StopReset()
	} else if voice != nil {
		voice.Stop()
	}
}

func DestroyVoice(voice Voice) {
	if destroy, ok := voice.(interface{ Destroy() }); ok {
		destroy.Destroy()
	} else if voice != nil {
		voice.Stop()
	}
}
