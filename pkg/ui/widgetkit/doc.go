// Package widgetkit holds the widgets the starter shares with the game: the
// flat panel painter, the check box marks, the text input model, the list
// wheel step, the window row, the window icon and the text clipboard. Each is
// the one builder of its kind for every program. The package imports neither
// pkg/ui, pkg/game nor pkg/sim, so a program that needs only these widgets
// does not link the game.
package widgetkit
