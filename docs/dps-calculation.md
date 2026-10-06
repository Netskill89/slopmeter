# DPS calculation

SlopMeter displays **player damage / max(fight seconds, 1)**. Every player uses
one shared fight clock; joining late does not give a player a shorter denominator.
Skill damage reconciles with player damage, and contribution is each player's
fraction of the group's captured damage. Refresh rate changes do not change totals.

The clock starts at the first accepted positive hit. During a live boss fight,
elapsed time includes mechanics and damage downtime. Finished fights freeze at
the encounter's end; ordinary fights end at the last damage timestamp after the
idle timeout. The reset button saves the interrupted fight, clears damage and
timing, and starts a new calculation on the next hit without restarting capture.

## Reference comparison

- [A2Tools DPS calculation](https://github.com/taengu/A2Tools-DPS-Meter/blob/856dad339be27099f052205b0ff3e2ddfcb77363/src-tauri/src/combat/dps_calculator.rs)
  uses damage / max(battle milliseconds, 1000) × 1000. Its selected-target window
  runs from first to last damage; its live number can therefore differ during
  downtime. SlopMeter deliberately uses elapsed boss-fight time.
- [Aion2Flow metric projection](https://github.com/cloris-chan/Aion2Flow/blob/8c7c3f3ce7770382afc646e74a5d97bca79b24aa/src/Aion2Flow.SceneRuntime/Projection/SceneCombatSnapshotAdapter.cs)
  divides damage by a shared encounter window in milliseconds. A zero-length
  window yields zero there; SlopMeter uses the one-second minimum above.

For matching damage and a ten-second window, a player dealing 10,000 damage
shows 1,000 DPS; another dealing 2,000 shows 200 DPS, even if they joined late.
Regression tests cover those values, contributions, skill totals, periodic damage,
long boss downtime and manual resets through the running capture process.

This is a comparison of source formulas and controlled replays, not a live
side-by-side benchmark. Decoding coverage can still differ: SlopMeter accepts
supported direct-hit layouts and allowlisted periodic damage. Unverified packet
forms and summon attribution can cause differences from other meters. Healing,
buffs and damage to self or confirmed group members are excluded.
