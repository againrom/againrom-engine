# Supplies the body sheet data/bodies.toml describes and draws the War Hammer
# with it. The two pictures under bodies/ are drawn by the test that uses this
# folder; they are not kept here.
def init(game, settings):
    game.data.add("data/bodies.toml")
    game.data.add("data/weapon-bodies.toml")
