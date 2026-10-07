package sim

// NativeCastContinuations counts complete native action records, including
// queued casts and recovery. These are distinct from retained SAV Effect or
// Projectile records; their presence cannot be inferred from presentation
// events or just the current actor facing.
func (w *World) NativeCastContinuations() (books, scrolls, scripts int) {
	return len(w.bookCasts), len(w.scrollCasts), len(w.casts)
}
