# Sound Options playback

The player can choose a track from the current screen's bank, play or stop
it, and switch between sequential and random order. The installed tunes table
supplies EN/RU titles. Selecting a row does not interrupt the current track.
Play uses that row, including rows reached by scrolling; repeated Play on the
current track preserves its decoder. Stop survives leaving the menu and a
cold application start. Playback preferences remain separate from the three
channel gains and master mute.

Authority: owner direction, VIDEO-MUSIC-056 and VIDEO-OPTIONS-057 in knowledge
k45. The list, Play/Stop and shuffle shape adopt their bounded consumers.
DIV-1296 retains original RNG and stop-transition debt. DIV-1297 records the
expanded layout, local preference format and remaining native UI unknowns.

Pointer, wheel, arrow keys and Home/End navigate the list. Tab moves between
controls. Missing manual tracks stay silent and report a refusal. Preference
writes complete before live changes; a failed write leaves playback intact.
One decoded track remains the cache bound. None of this changes simulation or
save fields.

Focused controller tests cover manual selection, sequential EOF, shuffle
changes, same-track Play, Stop, invalid/missing tracks and failed writes.
Profile tests cover defaults, persistence and unknown-key retention. The
installed witness selects B03 with the pointer and B10 by keyboard, checks
native PCM, Random, menu exit and cold Stop. The executable sound witness also
checks concrete device activity and leaves a stopped profile for a second
process. Paired release gates, exact executable receipts and the sole fresh
review are recorded by the seat before landing.
