package cryptops

import (
	"fmt"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/node"
	"gitlab.com/privategrity/server/services"
	"strconv"
	"testing"
)

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

	round := node.NewRound(batchSize)

	rng := cyclic.NewRandom(cyclic.NewInt(2), cyclic.NewInt(1000))

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

	node.InitUserRegistry()
	var users []node.User
	// make 1 more user than batchSize so that userID and slotID aren't the
	// same
	for i := uint64(0); i < batchSize+1; i++ {
		userAddress := strconv.FormatUint(i, 10)
		users = append(users, node.NewUser(userAddress))
	}

	users[1].Reception.BaseKey = cyclic.NewIntFromString(
		"da9f8137821987b978164932015c105263ae769310269b510937c190768e2930",
		16)
	users[2].Reception.BaseKey = cyclic.NewIntFromString(
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		16)
	users[3].Reception.BaseKey = cyclic.NewIntFromString(
		"f12345ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		16)

	users[1].Reception.RecursiveKey = cyclic.NewIntFromString(
		"ef9ab83927cd2349f98b1237889909002b897231ae9c927d1792ea0879287ea3",
		16)
	users[2].Reception.RecursiveKey = cyclic.NewIntFromString(
		"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		16)
	users[3].Reception.RecursiveKey = cyclic.NewIntFromString(
		"f12345ffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		16)

	for i := 0; i < len(users); i++ {
		node.UpsertUser(users[i])
	}

	var inSlots []services.Slot

	for i := uint64(0); i < batchSize; i++ {
		inSlots = append(inSlots, GenericKeySlot{slotID: i,
			userID: users[i+1].Id,
			// Run() currently gets the pair of keys from the user
			key:     cyclic.NewMaxInt(),
			keyType: RECEPTION,
		})
	}

	expectedRecursiveKeys := []*cyclic.Int{
		cyclic.NewIntFromString("5577eca469086dd710d29d28117d7014c0eb"+
			"bfb28fe488c2a2297e33f5dc6441", 16),
		cyclic.NewIntFromString("a957f9b2863d5575eb23092f846a8addd669"+
			"b6caaec02df65b2d174648f28179", 16),
		cyclic.NewIntFromString("2e8f49546c705902481bb56eaff3ac1e3124"+
			"84dc5c344e69ad50a8c7cbd62745", 16)}

	expectedSharedKeys := []*cyclic.Int{
		cyclic.NewIntFromString("4de2e3e634b726d210afd0284a2cac0653eb"+
			"37a347b34aac683b335653fd28fb4ca84073c1ce65ed54b04c64"+
			"e8033635b40ebbc0c2ce6e406ed8f6889a9e28920fef9c048ed9"+
			"2cddae1e8859afc77da66a7a7cd77932c52efa20c2fc1bc848f4"+
			"4e541874062692afeaeb17d62bdbc75e34681274fe129b8ea7be"+
			"d6a980cea7746e865af076c48de8b19940253a6ee658aed316af"+
			"fa291c5385b51b1105855a4aaf2c362b16f209f8ca45486dcc25"+
			"7cb2d88f9ad662c33bc1e9ac27e23e193ca38e2b560dd18a4b3d"+
			"a22e668bbc54d7820f297ceebacbb22003ae87b85c534498f7dd"+
			"c039cd991246d9d0352d8cedc3218c8f729488858cf3dfa6147"+
			"003c1948a68b65e55e73cdc8f81eaf5780c85b7bc6beb3ad3cf"+
			"36caaaca464804d2409f936c997909e29de89808a42e1450801"+
			"2c5e06a4449b396aa01eba6ea8b563a55df3f43d472ea2aec70"+
			"78a9914ab391d0032c59abbd1eeb65d03bc532ac6130f830ed29"+
			"380a0b2d40f40f33e7acfb739f243fcc7f8070186641f6e1b8d9"+
			"e3e051663521a4b1e8982337b3ea818d2df6aeb3256d118ba6b2"+
			"695301c81f1230d057ca1fdcbfd5205580e4ca71b190548f88c7"+
			"b058c1dee515bbde6c6b4eb7c78ddb1fd5d63b224be2d8b066b0"+
			"d00744365fde76f086992a942a669881e4302615c6d4c4204c07"+
			"bb05766cbcd8cbac7ac0d862a3f5e02036543af53684c63412b6"+
			"9b8f", 16),
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
		cyclic.NewIntFromString("432c9cf2b1e666cf3f72416db2d12ac8d27d"+
			"72ed9e9d8cd8e5cbf549916c5c54f4a9a400da0a868f78be9278"+
			"93195eaafaa2e7f9c0bfac873bc4686885e0db627bfd8d442452"+
			"534ffd32cd9560449ae2d1598758d006edfbea58457fc3f12467"+
			"a15ac7ac27d307376da3789d5ba3f15bb16cf0f025f602f49fb7"+
			"c6311d98042317da8f7db4b5de8b455e7ff9653e15ebac51754f"+
			"33a79b70441fcbc6deef1c4552fb3bc6989f05b84fb12acd9c89"+
			"b7481f20093127763580e87ec80c445f91fcec4b5d91fc104e1f"+
			"861aaef37bacc37409535be7d989db1af60a7167e82e166cc7dc"+
			"30b6a025f09acffe7cbcf1d4f42312ae84bfa5053e065b0e0b5e"+
			"966a2b4c976baf20359ba8dd86030bb149f36798f414249dcea5"+
			"9002c848ffea6b5db4ba9aeba7bd27602a0557509b29499ccc44"+
			"e7e558073b83d457c585d6673d441a053e865943629b8744e0d3"+
			"01af2fdbdb81c89d53387bab9fad701e20cabc12e97b2b03ddd1"+
			"5227fe045307a7733ff0ca5fad1327ff12d57bafccfd1a3ce796"+
			"d0a871978d8797190fcfbbca6a4c511159aebc51651d39e976b0"+
			"63cce4c0f35f8cc29f5f260e5e3304d8bd64091212adb3a6eac4"+
			"4cb7d5df7fb8b10dc7eb9805ecfdb548b92aed30c5b2ea07e8d8"+
			"aed6554b2b002737ea9a4b52355168c70c2ddf7499ad567bf659"+
			"d6f00f192f5cee0fc4f8cc343b6aec2a583dee78cc9eba762de1",
			16)}

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(inSlots[i])
		testOK := true
		actual := (*<-dc.OutChannel).(GenericKeySlot)
		if users[i+1].Reception.RecursiveKey.Cmp(expectedRecursiveKeys[i]) != 0 {
			testOK = false
			t.Error("Recursive keys differed at index", i)
		} else if actual.Key().Cmp(expectedSharedKeys[i]) != 0 {
			testOK = false
			t.Error("Shared keys differed at index", i)
		}

		if testOK {
			pass++
		}
	}

	println("Generate Client Key", pass, "out of", test, "tests passed.")

	for i := 0; i < len(users); i++ {
		node.DeleteUser(users[i].Id)
	}

	t.Errorf("Test not finished yet")
}
