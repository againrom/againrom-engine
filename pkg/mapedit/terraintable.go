package mapedit

// shapeTable names, for a cell on the primary or the secondary side of a strip
// and for water, the shape it shows given the mask of neighbours of the other
// class (bits NW, N, NE, W, E, SW, S, SE = 1, 2, 4, 8, 16, 32, 64, 128): one
// digit per mask, the Shape.Code of the commonest installed shape, or the fallback
// rule's where fewer than eight installed cells show the mask (DIV-3199).
var shapeTable = struct{ primary, secondary, water string }{
	"0065006145551011" +
		"8161606131111111" +
		"0040006074440011" +
		"3065606632540221" +
		"8444202374447001" +
		"8800202138556561" +
		"7044700374447734" +
		"3707051734744664" +
		"0025000035550065" +
		"8820202232844421" +
		"0000000000054088" +
		"0060006732844271" +
		"8828002330018661" +
		"8823282232852520" +
		"3011031333343634" +
		"3335352233343330",
	"0000000000dd00dd" +
		"00ee00ee000d00ed" +
		"0000000000dd00dd" +
		"00ee0eee00fd00ee" +
		"00000000ff0dfded" +
		"gg0dgeee0d0deded" +
		"00000f00fffdffff" +
		"gffdgeeefffdffef" +
		"0000000000dd0ded" +
		"00ee00ee00gd00ed" +
		"000d00e000fdfded" +
		"00gd00ee00fd00ed" +
		"00000g00ffgdgded" +
		"ggggggeggggdgged" +
		"00000000fffdfffd" +
		"ggggggegggfdfge0",
	"0511651144994499" +
		"2aaa22aa44994499" +
		"7591659144994499" +
		"22aa22aa44994499" +
		"3b113311bb99bb99" +
		"ccaaccaabb99bb99" +
		"3311c311bb99bb99" +
		"ccaaccaabb99bb99" +
		"85a16511bb994499" +
		"22aa22aa44994499" +
		"7511651144994499" +
		"22aa22aa44994499" +
		"33113311bb99bb99" +
		"ccaaccaabb99bb99" +
		"33113311bb99bb99" +
		"ccaaccaabb99bb99",
}
