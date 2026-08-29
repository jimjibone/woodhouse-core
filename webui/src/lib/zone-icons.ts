import type { Component } from 'svelte';
import SofaIcon from '@lucide/svelte/icons/sofa';
import ArmchairIcon from '@lucide/svelte/icons/armchair';
import TvIcon from '@lucide/svelte/icons/tv';
import LampIcon from '@lucide/svelte/icons/lamp';
import LampDeskIcon from '@lucide/svelte/icons/lamp-desk';
import LightbulbIcon from '@lucide/svelte/icons/lightbulb';
import PianoIcon from '@lucide/svelte/icons/piano';
import MusicIcon from '@lucide/svelte/icons/music';
import Gamepad2Icon from '@lucide/svelte/icons/gamepad-2';
import UtensilsIcon from '@lucide/svelte/icons/utensils';
import UtensilsCrossedIcon from '@lucide/svelte/icons/utensils-crossed';
import CookingPotIcon from '@lucide/svelte/icons/cooking-pot';
import RefrigeratorIcon from '@lucide/svelte/icons/refrigerator';
import MicrowaveIcon from '@lucide/svelte/icons/microwave';
import BedIcon from '@lucide/svelte/icons/bed';
import BabyIcon from '@lucide/svelte/icons/baby';
import BookOpenIcon from '@lucide/svelte/icons/book-open';
import BathIcon from '@lucide/svelte/icons/bath';
import ShowerHeadIcon from '@lucide/svelte/icons/shower-head';
import ToiletIcon from '@lucide/svelte/icons/toilet';
import ShirtIcon from '@lucide/svelte/icons/shirt';
import WashingMachineIcon from '@lucide/svelte/icons/washing-machine';
import MonitorIcon from '@lucide/svelte/icons/monitor';
import LaptopIcon from '@lucide/svelte/icons/laptop';
import BriefcaseIcon from '@lucide/svelte/icons/briefcase';
import DumbbellIcon from '@lucide/svelte/icons/dumbbell';
import DoorOpenIcon from '@lucide/svelte/icons/door-open';
import DoorClosedIcon from '@lucide/svelte/icons/door-closed';
import HouseIcon from '@lucide/svelte/icons/house';
import Building2Icon from '@lucide/svelte/icons/building-2';
import WarehouseIcon from '@lucide/svelte/icons/warehouse';
import CarIcon from '@lucide/svelte/icons/car';
import CaravanIcon from '@lucide/svelte/icons/caravan';
import BikeIcon from '@lucide/svelte/icons/bike';
import HammerIcon from '@lucide/svelte/icons/hammer';
import WrenchIcon from '@lucide/svelte/icons/wrench';
import PackageIcon from '@lucide/svelte/icons/package';
import ArchiveIcon from '@lucide/svelte/icons/archive';
import TreesIcon from '@lucide/svelte/icons/trees';
import FlowerIcon from '@lucide/svelte/icons/flower';
import SproutIcon from '@lucide/svelte/icons/sprout';
import FenceIcon from '@lucide/svelte/icons/fence';
import TentIcon from '@lucide/svelte/icons/tent';
import SunIcon from '@lucide/svelte/icons/sun';
import DogIcon from '@lucide/svelte/icons/dog';
import CatIcon from '@lucide/svelte/icons/cat';
import MapPinIcon from '@lucide/svelte/icons/map-pin';
import LayoutGridIcon from '@lucide/svelte/icons/layout-grid';

export type ZoneIcon = {
	key: string;
	label: string;
	icon: Component;
};

// The icons an admin can pick for a zone.
//
// The server stores the key as an opaque string and never looks at it, so this
// list is purely a client concern - it keeps the picker to icons that actually
// read as rooms and areas, and keeps the bundle to these icons rather than the
// whole Lucide catalogue (hence the per-icon deep imports above).
//
// Removing an entry is safe: a zone still holding that key falls back to the
// default rather than failing to render.
export const ZONE_ICONS: ZoneIcon[] = [
	{ key: 'sofa', label: 'Sofa', icon: SofaIcon },
	{ key: 'armchair', label: 'Armchair', icon: ArmchairIcon },
	{ key: 'tv', label: 'TV', icon: TvIcon },
	{ key: 'lamp', label: 'Lamp', icon: LampIcon },
	{ key: 'lamp-desk', label: 'Desk Lamp', icon: LampDeskIcon },
	{ key: 'lightbulb', label: 'Light', icon: LightbulbIcon },
	{ key: 'piano', label: 'Piano', icon: PianoIcon },
	{ key: 'music', label: 'Music', icon: MusicIcon },
	{ key: 'gamepad-2', label: 'Games', icon: Gamepad2Icon },
	{ key: 'utensils', label: 'Utensils', icon: UtensilsIcon },
	{ key: 'utensils-crossed', label: 'Dining', icon: UtensilsCrossedIcon },
	{ key: 'cooking-pot', label: 'Cooking', icon: CookingPotIcon },
	{ key: 'refrigerator', label: 'Fridge', icon: RefrigeratorIcon },
	{ key: 'microwave', label: 'Microwave', icon: MicrowaveIcon },
	{ key: 'bed', label: 'Bed', icon: BedIcon },
	{ key: 'baby', label: 'Nursery', icon: BabyIcon },
	{ key: 'book-open', label: 'Books', icon: BookOpenIcon },
	{ key: 'bath', label: 'Bath', icon: BathIcon },
	{ key: 'shower-head', label: 'Shower', icon: ShowerHeadIcon },
	{ key: 'toilet', label: 'Toilet', icon: ToiletIcon },
	{ key: 'shirt', label: 'Wardrobe', icon: ShirtIcon },
	{ key: 'washing-machine', label: 'Laundry', icon: WashingMachineIcon },
	{ key: 'monitor', label: 'Desktop', icon: MonitorIcon },
	{ key: 'laptop', label: 'Laptop', icon: LaptopIcon },
	{ key: 'briefcase', label: 'Office', icon: BriefcaseIcon },
	{ key: 'dumbbell', label: 'Gym', icon: DumbbellIcon },
	{ key: 'door-open', label: 'Hallway', icon: DoorOpenIcon },
	{ key: 'door-closed', label: 'Door', icon: DoorClosedIcon },
	{ key: 'house', label: 'House', icon: HouseIcon },
	{ key: 'building-2', label: 'Building', icon: Building2Icon },
	{ key: 'warehouse', label: 'Garage', icon: WarehouseIcon },
	{ key: 'car', label: 'Car', icon: CarIcon },
	{ key: 'caravan', label: 'Caravan', icon: CaravanIcon },
	{ key: 'bike', label: 'Bike', icon: BikeIcon },
	{ key: 'hammer', label: 'Workshop', icon: HammerIcon },
	{ key: 'wrench', label: 'Utility', icon: WrenchIcon },
	{ key: 'package', label: 'Store Room', icon: PackageIcon },
	{ key: 'archive', label: 'Loft', icon: ArchiveIcon },
	{ key: 'trees', label: 'Garden', icon: TreesIcon },
	{ key: 'flower', label: 'Flowers', icon: FlowerIcon },
	{ key: 'sprout', label: 'Greenhouse', icon: SproutIcon },
	{ key: 'fence', label: 'Yard', icon: FenceIcon },
	{ key: 'tent', label: 'Outdoors', icon: TentIcon },
	{ key: 'sun', label: 'Patio', icon: SunIcon },
	{ key: 'dog', label: 'Dog', icon: DogIcon },
	{ key: 'cat', label: 'Cat', icon: CatIcon },
	{ key: 'map-pin', label: 'Pin', icon: MapPinIcon },
	{ key: 'layout-grid', label: 'Grid', icon: LayoutGridIcon },
];

const byKey = new Map(ZONE_ICONS.map((entry) => [entry.key, entry.icon]));

export const DEFAULT_ZONE_ICON_KEY = 'map-pin';

// Resolve a stored icon key to a component. Unknown keys - an icon dropped from
// the list above, or a zone created by another client - render the default
// rather than blowing up the nav.
export function zoneIcon(key: string): Component {
	return byKey.get(key) ?? MapPinIcon;
}
