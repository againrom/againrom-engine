# Adds a sellsword to the character catalog and a chain of two missions the
# guests play before the party reaches the town: data/characters.toml and
# data/missions.toml.
def init(game, settings):
    game.data.add("data/characters.toml")
    game.data.add("data/missions.toml")
