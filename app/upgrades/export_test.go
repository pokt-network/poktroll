package upgrades

// SeedServiceCapParams exposes seedServiceCapParams to the external test package,
// which cannot live in package upgrades (testutil/keeper imports app, which imports upgrades).
var SeedServiceCapParams = seedServiceCapParams
