package cryptops

import (
	"fmt"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"strconv"
	"testing"
)

// GenericKeySlot implements the KeySlot interface in the simplest way
// possible. It's not meant for use outside testing.
type GenericKeySlot struct {
	slotID  uint64
	userID  uint64
	key     *cyclic.Int
	keyType KeyType
}

func (g GenericKeySlot) SlotID() uint64 {
	return g.slotID
}

func (g GenericKeySlot) UserID() uint64 {
	return g.userID
}

func (g GenericKeySlot) Key() *cyclic.Int {
	return g.key
}

func (g GenericKeySlot) GetKeyType() KeyType {
	return g.keyType
}

func TestGenerateClientKey(t *testing.T) {
	// NOTE: Does not test correctness

	test := 3
	pass := 0

	batchSize := uint64(3)

	round := globals.NewRound(batchSize)

	rng := cyclic.NewRandom(cyclic.NewInt(2), cyclic.NewInt(1000))

	// This prime is 4096 bits
	prime := cyclic.NewIntFromString(
		"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64"+
			"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7"+
			"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B"+
			"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C"+
			"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31"+
			"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7"+
			"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA"+
			"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6"+
			"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED"+
			"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9"+
			"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199"+
			"FFFFFFFFFFFFFFFF", 16)

	group := cyclic.NewGroup(prime, cyclic.NewInt(55), cyclic.NewInt(33), rng)

	dc := services.DispatchCryptop(&group, GenerateClientKey{}, nil, nil, round)

	// Create user registry, where Run() gets its pair of keys.
	var users []*globals.User
	// Make 1 more user than batchSize so that userID and slotID aren't the
	// same. This should ensure that userID is used where it should be and
	// slotID is used where it should be.
	for i := uint64(0); i < batchSize+1; i++ {
		userAddress := strconv.FormatUint(i, 10)
		users = append(users, globals.Users.NewUser(userAddress))
	}

	users[1].Reception.BaseKey = cyclic.NewIntFromString(
		"da9f8137821987b978164932015c105263ae769310269b510937c190768e2930",
		16)
	users[2].Reception.BaseKey = cyclic.NewIntFromString(
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		16)
	users[3].Reception.BaseKey = cyclic.NewIntFromString(
		"ef9ab83927cd2349f98b1237819909002b897231ae9c927d1792ea0879287ea3",
		16)

	users[1].Reception.RecursiveKey = cyclic.NewIntFromString(
		"ef9ab83927cd2349f98b1237889909002b887231ae9c927d1792ea0879287ea3",
		16)
	users[2].Reception.RecursiveKey = cyclic.NewIntFromString(
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		16)
	users[3].Reception.RecursiveKey = cyclic.NewIntFromString(
		"da9f8137821987b978164932015c105263ae799310269b510937c190768e2930",
		16)

	for i := 0; i < len(users); i++ {
		globals.Users.UpsertUser(users[i])
	}

	var inSlots []services.Slot

	for i := uint64(0); i < batchSize; i++ {
		inSlots = append(inSlots, GenericKeySlot{slotID: i,
			userID:  users[i+1].Id,
			key:     cyclic.NewMaxInt(),
			keyType: RECEPTION,
		})
	}

	// These expected keys were generated by the test
	expectedRecursiveKeys := []*cyclic.Int{
		cyclic.NewIntFromString("32cef0d857f07ea3f8b4658b0288bcccd324"+
			"3677413213070a9e20d5d51857fb", 16),
		cyclic.NewIntFromString("a957f9b2863d5575eb23092f846a8addd669"+
			"b6caaec02df65b2d174648f28179", 16),
		cyclic.NewIntFromString("467861f48da75ffaeada0b017736e220f70f"+
			"32e9629fff9b7bc3709137d77623", 16)}

	expectedSharedKeys := []*cyclic.Int{
		cyclic.NewIntFromString("bf35b235dafab1f38e72694d3471f1679386"+
			"69eba4833d6909a60dca4abac4d1a7601a77d98a74280aee7858"+
			"dc27732840703973e5299e5e9a95427414fdbacf3a7267c3d522"+
			"1e2d74fe1f21d380d08671d27633e1e93073c541ee9be12f588d"+
			"53dc168477653f1184ae86cf482557253fcffc5bceed574e923c"+
			"64f4b78c9646daae02eb2ce5769e479047aea35e3c84db395de0"+
			"7ed9c52fcbb4158f16e88068a1023e286337102118f7973f7891"+
			"64ef6e3d8c554ba53b3db00f3f1a55d3cc809ae39ad9c78a9196"+
			"9102d2fecdbfc1436b74589dd09a57b852e84ed8548cf248d188"+
			"76608669d16889fed51ec59fb0ad2599f964ee204bfcebc697cf"+
			"0ec96aeff55496972c2e3b03612dc62328ef10bdadde3be6f824"+
			"76997b0d770956a604dc7c87e96cfa6f2fc246b4b28ed5ec013b"+
			"9685a0a84e1fa89b585e84418a3bf3d096947fd403b2de7d6a99"+
			"7f4f03b55677145566ff208ab637ede55e910eacf07d9696d167"+
			"975163d913d655db0e7ff8749e619189ccad766453286d428bae"+
			"dd688e74cb720ea1b587b8583fca72ea17938b9084848dbf2227"+
			"a86b6cb66c000e2e3dd58e0172556e8f2bd96809eb78638bd4d3"+
			"385e7c342ad01cec0f1a8cd9cfcb0d6174c2650cf7fe95ed8e08"+
			"69b535bd6f68e3b82190882394d85d620ac0f04d879e8b194756"+
			"e72c4f4c1ae59dae4d7c75fdad877769a2cfce26fb7f81968cb1",
			16),
		cyclic.NewIntFromString("a36332dd511ea27fce9880dbc9a0b8975d9e"+
			"1bf19e572e0416f8de826ca3675364c5731859257b9fccf49880"+
			"2f32346a40b27f9d7c84079e0bac37e59dd2bacfa55dc5c5d622"+
			"561c800356c17cebeffd32b7899113c00b7e0f5c8941a959d372"+
			"41a9e2ff3856a3d48430f4b75235d8eee250f23a7a357b6d973e"+
			"d80b3bc957bca3425aa1cce3cfd6cfe515f30f7a939a5e75d83b"+
			"f9eed5a92d95e2cea44099e06e8a1e64afe285f7ac1c2c03cef2"+
			"65c396a1d4a8f20a622b819a3ca330e32790f1ab67e8ed5fd19f"+
			"9d597f58d7fdeb8033b48fcbccb8f598f4dd107c85175ff93df2"+
			"3dc6945f55d22018489e66bca784cefb7edcdc372d1767065f7f"+
			"33bd14bca4adf66665a42b2110e1f3bee8ecbf5a460379c5dc90"+
			"f56748bfaf1c24a3548f8dd23656afd7dcfef3ffa5e034b30bf0"+
			"9130a198770f9792b47c254f74162588075c0afbcf98659318e2"+
			"6a7c57e2ae3d5c087a22cb69c6b7f65c4b40559bdf1d94c855c5"+
			"2a04d2fcd53c7babfd598a46609cd2e6a6fc86594fc5d9add1f1"+
			"65928cb516b31bda78b06eed85cc85d13134367bfd65e2b77a2c"+
			"c2744f78d30574f382cbc7cce28c1e54ebc7dab49885d2b6d8f9"+
			"1879caac3e3cc0590691aa2e38af10651ded8e2436f99dc826a6"+
			"7684bd39e2f2e4d0264f4a1d1f6ee389e3c56ef0b74cfb6f9403"+
			"42f9fecdd7edc950d6b3de3ae567313cfe9c7e37e55c1e9ad8e3",
			16),
		cyclic.NewIntFromString("a71da7b470a9040a7189cdd231ce82ea5314"+
			"7eada52b1a5e8ee05443c3e411c20f65502dbb011a3bde18b03d"+
			"40426b15156430dec01f3e2fc297b08358546f4ed746e59f002e"+
			"e93dc5774c397a533810b01de7844015b89da6b495ca9a5135dc"+
			"079fbe4e4bfe87f057c931e7881694cc79a0990716606a02c732"+
			"cd4165d63e0c82cec3fe951960dc38054936760d8b1bd0bdaa3d"+
			"a7e9260374ddc1b1b0746b58b1792c97e11108daf1725511a79d"+
			"e3a8026bc87011b8b7142e7cffc79da9938ad40d4ffcc6afe662"+
			"72ac7db6f9c265fbc3eecf303a84efb4052f60910010a1de5c38"+
			"ccd5512ee89a6cad3d22df8697537eda4441b2610d9622aa0364"+
			"94153c569c4b4914a1c606016bde9114c91d9cceaee1064acef8"+
			"19680acbc53f10a42629239cfac06e94af52521bccf6de0bd137"+
			"d194356a1bf95778af4174bd39df7a6cc3a64024e09c6c875bf9"+
			"f50e323bc5cb63e1b367dcbf0ddeb9d36552cc03ab4965a63b8e"+
			"a622497961c22141bdf0beb6e8cdb60e95a471c77390b914a817"+
			"cca08aa9add9145ef3a682be8092cacf9c4e7429bd44d06bca9c"+
			"3f809c7601c6225385cb24496279dfabdc2ec98f6bcd494e42b1"+
			"a2f40d1bb33b8c2035ddd51d6c490cf14c87b4651a0de72a34ed"+
			"5287f70913af91eb8a699afed8b99c6af7fa901707205bc22058"+
			"b46a2be8381248ed71ab731fe85ea734c3ee581144a6e0ffba2c",
			16)}

	// Do the test
	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(inSlots[i])
		testOK := true
		actual := (*<-dc.OutChannel).(GenericKeySlot)
		if users[i+1].Reception.RecursiveKey.Cmp(expectedRecursiveKeys[i]) != 0 {
			testOK = false
			t.Error("Recursive keys differed at index", i)
		} else if actual.Key().Cmp(expectedSharedKeys[i]) != 0 {
			fmt.Println(actual.Key().Text(16))
			testOK = false
			t.Error("Shared keys differed at index", i)
		}

		if testOK {
			pass++
		}
	}

	println("Generate Client Key", pass, "out of", test, "tests passed.")

	// Clean up user registry
	for i := 0; i < len(users); i++ {
		globals.Users.DeleteUser(users[i].Id)
	}
}
