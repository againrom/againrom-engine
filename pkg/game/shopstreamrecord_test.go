package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/random"
)

func shopStreamState(t *testing.T, service *random.Service) string {
	t.Helper()
	stream := service.Stream(random.ShopStock)
	if stream.Shared() {
		return fmt.Sprintf("shared %d", service.SharedState())
	}
	h := sha256.New()
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		fmt.Fprintf(h, "%s %s;", v.Type(), v.Kind())
		switch v.Kind() {
		case reflect.Pointer, reflect.Interface:
			if v.IsNil() {
				fmt.Fprint(h, "nil;")
			} else {
				visit(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i))
			}
		case reflect.Array, reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fmt.Fprintf(h, "%d;", v.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			fmt.Fprintf(h, "%d;", v.Uint())
		case reflect.Bool:
			fmt.Fprintf(h, "%t;", v.Bool())
		default:
			t.Fatalf("unmeasured RNG field %s %s", v.Type(), v.Kind())
		}
	}
	visit(reflect.ValueOf(stream).Elem().FieldByName("g"))
	return fmt.Sprintf("seeded %x", h.Sum(nil))
}

func TestShopStreamStateRecord(t *testing.T) {
	var out strings.Builder
	for _, mode := range []random.Mode{random.Seeded, random.Original} {
		f := shellFrontEnd()
		f.Table = shopTable()
		f.Random = random.NewService(random.Session{Seed: 71, Mode: mode})
		screen := f.townUI
		screen.draws = runtimeTownDraws{&f.RuntimeServices}
		f.Shop = NewShop(1000)
		f.Shop.GenerateWith(f.Table, 7, f.Random.Stream(random.ShopStock))
		line := func(game, event string, svc *random.Service) {
			fmt.Fprintf(&out, "%s mode %d %s: %s\n", game, mode, event, shopStreamState(t, svc))
		}
		line("first", "arrival", f.Random)
		screen.bindShop(screen.chapterShopRow())
		line("first", "entry", f.Random)
		for n := 1; n <= 3; n++ {
			if !screen.sleep() {
				t.Fatal("first Sleep")
			}
			line("first", fmt.Sprintf("sleep %d", n), f.Random)
		}
		f.Shop = NewShop(25)
		f.Shop.GenerateWith(f.Table, shopSeed(2, 2, 25), f.Random.Stream(random.ShopStock))
		line("first", "chapter", f.Random)
		for _, id := range []int{1, 2, 3} {
			f, screen := sleepTown(id)
			f.Random = random.NewService(random.Session{Seed: 71, Mode: mode})
			tables, _ := secondShopTables(f.Table)
			c := f.Town.second
			c.shop.drawStock(tables, id, f.Random.Stream(random.ShopStock).Raw)
			game := fmt.Sprintf("second %d", id)
			line(game, "arrival", f.Random)
			c.shop.enter(tables, id, f.Random.Stream(random.ShopStock).Raw, c.heroBag())
			line(game, "entry", f.Random)
			for n := 1; n <= 3; n++ {
				if !screen.sleep() {
					t.Fatal("second Sleep")
				}
				line(game, fmt.Sprintf("sleep %d", n), f.Random)
			}
			c.shop.depart(50)
			c.shop.stage(90)
			line(game, "record writes", f.Random)
			c.shop.enter(tables, 3, f.Random.Stream(random.ShopStock).Raw, c.heroBag())
			line(game, "changed record entry", f.Random)
		}
	}
	path := filepath.Join("testdata", "shop_stream_record.txt")
	if os.Getenv("AGAINROM_WRITE_SHOP_STREAM_RECORD") == "1" {
		if err := os.WriteFile(path, []byte(out.String()), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != out.String() {
		t.Fatalf("full stock-stream record changed:\n%s\nwant:\n%s", out.String(), want)
	}
}
