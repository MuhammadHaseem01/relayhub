DO $$
BEGIN
	CREATE TYPE "Role" AS ENUM ('Super Admin', 'Admin', 'Manager', 'User');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "Status" AS ENUM ('Active', 'Inactive');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "DeductionType" AS ENUM ('Cancelled', 'Late Delivery');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "DamageType" AS ENUM ('Broken', 'Damaged Packaging', 'Missing Items', 'Scratched');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "LicenseType" AS ENUM ('HTV', 'PSV', 'IDP', 'LTV');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "DriverStatus" AS ENUM ('Avaliable', 'On Leave', 'Suspended', 'On Trip');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "VehicleStatus" AS ENUM ('Avaliable', 'On Trip', 'Maintenance', 'Out of Service');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "VehicleType" AS ENUM (
		'Flatbed Trailer',
		'Low Bed Trailer',
		'Container Trailer',
		'Skeletal Trailer',
		'Curtain Side Trailer',
		'Refrigerated Trailer (Reefer)',
		'Fuel Tanker',
		'Water Tanker',
		'Dump Truck / Tipper',
		'Car Carrier',
		'17 ft Mazda',
		'22 ft Truck',
		'28 ft Truck',
		'Tractor Head'
	);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "TourStatus" AS ENUM ('Pre-Planned', 'In Progress', 'Completed', 'Cancelled', 'Late Delivery');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "TripPaymentStatus" AS ENUM ('Received', 'Pending', 'Partial Received', 'Paid');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "VehicleMaintenanceCategory" AS ENUM ('Engine', 'Tires', 'Brakes', 'Electrical', 'Suspension', 'Transmission', 'Oil & Fluids', 'Body / General');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "VehicleMaintenancePaymentStatus" AS ENUM ('Paid', 'Pending');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "InventoryCategory" AS ENUM ('Spare', 'Fuel', 'Goods', 'Tools');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
	CREATE TYPE "InventoryUnitType" AS ENUM ('Litre', 'KG', 'Piece');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS "User" (
	"id" SERIAL PRIMARY KEY,
	"name" TEXT NOT NULL,
	"email" TEXT NOT NULL UNIQUE,
	"password" TEXT NOT NULL,
	"organizationName" TEXT NOT NULL,
	"organizationId" TEXT,
	"role" "Role" NOT NULL,
	"permissions" TEXT[] NOT NULL,
	"status" "Status" NOT NULL,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "Driver" (
	"id" SERIAL PRIMARY KEY,
	"fullName" TEXT NOT NULL,
	"phoneNumber" TEXT NOT NULL,
	"driverLicenseNumber" TEXT NOT NULL,
	"licenseType" "LicenseType" NOT NULL,
	"licenseExpiry" TIMESTAMP(3) NOT NULL,
	"status" "DriverStatus" NOT NULL,
	"statusDateTime" TIMESTAMP(3) NOT NULL,
	"email" TEXT NOT NULL,
	"address" TEXT NOT NULL,
	"emergencyContactName" TEXT NOT NULL,
	"emergencyContactPhone" TEXT NOT NULL,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "DriverStatusHistory" (
	"id" SERIAL PRIMARY KEY,
	"status" "DriverStatus" NOT NULL,
	"date" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"driverId" INTEGER NOT NULL REFERENCES "Driver"("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "Client" (
	"id" SERIAL PRIMARY KEY,
	"fullName" TEXT NOT NULL,
	"phoneNumber" TEXT NOT NULL,
	"email" TEXT NOT NULL,
	"address" TEXT NOT NULL,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "Vehicle" (
	"id" SERIAL PRIMARY KEY,
	"vehicleNumber" TEXT NOT NULL,
	"mtag" TEXT DEFAULT '0',
	"vehicleType" "VehicleType" NOT NULL,
	"status" "VehicleStatus" NOT NULL,
	"statusDateTime" TIMESTAMP(3) NOT NULL,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "VehicleStatusHistory" (
	"id" SERIAL PRIMARY KEY,
	"status" "VehicleStatus" NOT NULL,
	"date" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"vehicleId" INTEGER NOT NULL REFERENCES "Vehicle"("id") ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "ArrangeVehicle" (
	"id" SERIAL PRIMARY KEY,
	"vehicleNumber" TEXT NOT NULL,
	"vehicleType" TEXT NOT NULL,
	"ownerName" TEXT NOT NULL,
	"ownerPhoneNumber" TEXT NOT NULL,
	"email" TEXT,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "Inventory" (
	"id" SERIAL PRIMARY KEY,
	"itemName" TEXT NOT NULL,
	"category" "InventoryCategory" NOT NULL,
	"unitType" "InventoryUnitType" NOT NULL,
	"currentStockQty" DOUBLE PRECISION NOT NULL,
	"purchasePrice" DOUBLE PRECISION NOT NULL,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "VehicleMaintenance" (
	"id" SERIAL PRIMARY KEY,
	"vehicleId" TEXT NOT NULL,
	"vehicleName" TEXT,
	"workService" TEXT NOT NULL,
	"category" "VehicleMaintenanceCategory" NOT NULL,
	"date" TIMESTAMP(3) NOT NULL,
	"amount" DOUBLE PRECISION NOT NULL,
	"paymentStatus" "VehicleMaintenancePaymentStatus" NOT NULL,
	"vendorWorkshop" TEXT NOT NULL,
	"description" TEXT,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "Ledger" (
	"id" SERIAL PRIMARY KEY,
	"uid" TEXT NOT NULL,
	"sourceType" TEXT NOT NULL,
	"sourceId" TEXT NOT NULL,
	"sourceName" TEXT NOT NULL,
	"amount" DOUBLE PRECISION NOT NULL,
	"paymentStatus" TEXT NOT NULL,
	"date" TIMESTAMP(3) NOT NULL,
	"referenceType" TEXT NOT NULL,
	"referenceId" TEXT NOT NULL,
	"referenceModel" TEXT NOT NULL,
	"referenceRecordId" INTEGER NOT NULL,
	"lineIndex" INTEGER NOT NULL,
	"createdById" INTEGER,
	"organizationId" TEXT,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "TourDeduction" (
	"id" SERIAL PRIMARY KEY,
	"tourId" TEXT NOT NULL,
	"deductedAmount" DOUBLE PRECISION NOT NULL,
	"deductionType" "DeductionType" NOT NULL,
	"deductionDate" TIMESTAMP(3) NOT NULL,
	"reason" TEXT NOT NULL,
	"status" "DeductionType" NOT NULL,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "Tour" (
	"id" SERIAL PRIMARY KEY,
	"tourName" TEXT NOT NULL,
	"driver" TEXT,
	"vehicle" TEXT,
	"client" TEXT NOT NULL,
	"startLocation" JSONB NOT NULL,
	"endLocation" JSONB NOT NULL,
	"startDate" TIMESTAMP(3) NOT NULL,
	"time" TEXT NOT NULL,
	"expectedEndDate" TIMESTAMP(3) NOT NULL,
	"actualEndDate" TIMESTAMP(3),
	"actualEndTime" TEXT,
	"freightAmount" DOUBLE PRECISION NOT NULL,
	"advanceAmount" DOUBLE PRECISION,
	"otherCharges" DOUBLE PRECISION,
	"paymentStatus" "TripPaymentStatus" NOT NULL,
	"partialReceivedPayment" DOUBLE PRECISION,
	"expenses" JSONB NOT NULL,
	"fuelDetails" JSONB NOT NULL,
	"loadType" TEXT,
	"cargoWeight" DOUBLE PRECISION,
	"vehicleType" "VehicleType",
	"status" "TourStatus" NOT NULL,
	"notes" TEXT,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS "TourDamage" (
	"id" SERIAL PRIMARY KEY,
	"tourId" TEXT NOT NULL,
	"tourName" TEXT,
	"vehicleId" TEXT,
	"vehicleName" TEXT,
	"driverId" TEXT,
	"driverName" TEXT,
	"damageType" "DamageType" NOT NULL,
	"damageDate" TIMESTAMP(3) NOT NULL,
	"reason" TEXT,
	"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
	"createdById" INTEGER REFERENCES "User"("id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS "DriverStatusHistory_driverId_idx" ON "DriverStatusHistory" ("driverId");
CREATE INDEX IF NOT EXISTS "DriverStatusHistory_driverId_date_idx" ON "DriverStatusHistory" ("driverId", "date");
CREATE INDEX IF NOT EXISTS "VehicleStatusHistory_vehicleId_idx" ON "VehicleStatusHistory" ("vehicleId");
CREATE INDEX IF NOT EXISTS "VehicleStatusHistory_vehicleId_date_idx" ON "VehicleStatusHistory" ("vehicleId", "date");
CREATE INDEX IF NOT EXISTS "ArrangeVehicle_createdById_idx" ON "ArrangeVehicle" ("createdById");
CREATE INDEX IF NOT EXISTS "ArrangeVehicle_vehicleNumber_idx" ON "ArrangeVehicle" ("vehicleNumber");
CREATE INDEX IF NOT EXISTS "Inventory_createdById_idx" ON "Inventory" ("createdById");
CREATE INDEX IF NOT EXISTS "Inventory_itemName_idx" ON "Inventory" ("itemName");
CREATE INDEX IF NOT EXISTS "VehicleMaintenance_createdById_idx" ON "VehicleMaintenance" ("createdById");
CREATE INDEX IF NOT EXISTS "VehicleMaintenance_vehicleId_idx" ON "VehicleMaintenance" ("vehicleId");
CREATE INDEX IF NOT EXISTS "VehicleMaintenance_date_idx" ON "VehicleMaintenance" ("date");
CREATE UNIQUE INDEX IF NOT EXISTS "Ledger_reference_unique" ON "Ledger" ("referenceModel", "referenceRecordId", "lineIndex", "sourceType", "sourceId");
CREATE INDEX IF NOT EXISTS "Ledger_createdById_idx" ON "Ledger" ("createdById");
CREATE INDEX IF NOT EXISTS "Ledger_date_idx" ON "Ledger" ("date");
CREATE INDEX IF NOT EXISTS "Ledger_reference_idx" ON "Ledger" ("referenceModel", "referenceRecordId");
