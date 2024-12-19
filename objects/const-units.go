package objects

const (
	UnitSquareMetre uint16 = iota
	UnitSquareFoot
	UnitMilliampere
	UnitAmpere
	UnitOhm
	UnitVolt
	UnitKilovolt
	UnitMegavolt
	UnitVoltAmpere
	UnitKilovoltAmpere
	UnitMegavoltAmpere
	UnitVoltAmperesReactive
	UnitKilovoltAmpereReactive
	UnitMegavoltAmperesReactive
	UnitDegreesPhase
	UnitPowerFactor
	UnitJoule
	UnitKilojoule
	UnitWattHour
	UnitKilowattHour
	UnitBritishThermalUnitMean
	UnitThermUS
	UnitTonUSPerHour
	UnitJoulesPerKilogramDryAir
	UnitBtusPerPoundDryAir
	UnitCyclesPerHour
	UnitCyclesPerMinute
	UnitHertz
	UnitGramsOfWaterPerKilogramDryAir
	UnitPercentRelativeHumidity
	UnitMillimetre
	UnitMetre
	UnitInch
	UnitFoot
	UnitWattsPerSquareFoot
	UnitWattsPerSquareMeter
	UnitLumen
	UnitLux
	UnitFootcandle
	UnitKilogram
	UnitPoundsMass
	UnitTons
	UnitKilogramPerSecond
	UnitKilogramPerMinute
	UnitKilogramPerHour
	UnitPoundsMassPerMinute
	UnitPoundsMassPerHour
	UnitWatt
	UnitKilowatt
	UnitMegawatt
	UnitBritishThermalUnitThermochemicalPerHour
	UnitHorsepower
	UnitTonsRefrigeration
	UnitPascal
	UnitKilopascal
	UnitBarUnitOfPressure
	UnitPoundForcePerSquareInch
	UnitCentimetersOfWater
	UnitInchOfWater
	UnitMillimetersOfMercury
	UnitCentimetreOfMercury
	UnitInchOfMercury
	UnitDegreeCelsius
	UnitKelvin
	UnitDegreeFahrenheit
	UnitDegreeDaysCelsius
	UnitDegreeDaysFahrenheit
	UnitYears
	UnitMonth
	UnitWeek
	UnitDay
	UnitHour
	UnitMinuteUnitOfTime
	UnitSecondUnitOfTime
	UnitMetrePerSecond
	UnitKilometrePerHour
	UnitFootPerSecond
	UnitFootPerMinute
	UnitMilePerHourStatuteMile
	UnitCubicFoot
	UnitCubicMetre
	UnitImperialGallons
	UnitLitre
	UnitGallonUS
	UnitCubicFootPerMinute
	UnitCubicMetrePerSecond
	UnitImperialGallonPerMinute
	UnitLitrePerSecond
	UnitLitrePerMinute
	UnitUSGallonPerMinute
	UnitDegreeUnitOfAngle
	UnitDegreeCelsiusPerHour
	UnitDegreeCelsiusPerMinute
	UnitDegreeFahrenheitPerHour
	UnitDegreeFahrenheitPerMinute
	UnitNoUnits
	UnitPartsPerMillion
	UnitPartsPerBillion
	UnitPercent
	UnitPercentPerSecond
	UnitPerMinute
	UnitPerSecond
	UnitPsiPerDegreeFahrenheit
	UnitRadians
	UnitRevolutionsPerMinute
	UnitCurrency1
	UnitCurrency2
	UnitCurrency3
	UnitCurrency4
	UnitCurrency5
	UnitCurrency6
	UnitCurrency7
	UnitCurrency8
	UnitCurrency9
	UnitCurrency10
	UnitSquareInch
	UnitSquareCentimetre
	UnitBritishThermalUnitInternationalTablePerPound
	UnitCentimetre
	UnitPoundsMassPerSecond
	UnitDeltaDegreesFahrenheit
	UnitDeltaDegreesKelvin
	UnitKiloohm
	UnitMegaohm
	UnitMillivolt
	UnitKilojoulePerKilogram
	UnitMegajoule
	UnitJoulesPerDegreeKelvin
	UnitJoulesPerKilogramDegreeKelvin
	UnitKilohertz
	UnitMegahertz
	UnitPerHour
	UnitMilliwatt
	UnitHectopascal
	UnitMillibar
	UnitCubicMetrePerHour
	UnitLitersPerHour
	UnitKilowattHoursPerSquareMeter
	UnitKilowattHoursPerSquareFoot
	UnitMegajoulesPerSquareMeter
	UnitMegajoulesPerSquareFoot
	UnitWattsPerSquareMeterDegreeKelvin
	UnitCubicFeetPerSecond
	UnitPercentObscurationPerFoot
	UnitPercentObscurationPerMeter
	UnitMilliohm
	UnitMegawattHour1000KWh
	UnitKiloBtus
	UnitMegaBtus
	UnitKilojoulesPerKilogramDryAir
	UnitMegajoulesPerKilogramDryAir
	UnitKilojoulesPerDegreeKelvin
	UnitMegajoulesPerDegreeKelvin
	UnitNewton
	UnitGramPerSecond
	UnitGramPerMinute
	UnitTonnePerHour
	UnitKiloBtusPerHour
	UnitHundredthsSeconds
	UnitMillisecond
	UnitNewtonMetre
	UnitMillimetrePerSecond
	UnitMillimetrePerMinute
	UnitMetrePerMinute
	UnitMetrePerHour
	UnitCubicMetrePerMinute
	UnitMetrePerSecondSquared
	UnitAmperePerMetre
	UnitAmperePerSquareMetre
	UnitAmpereSquareMetre
	UnitFarad
	UnitHenry
	UnitOhmMetre
	UnitSiemens
	UnitSiemensPerMetre
	UnitTesla
	UnitVoltPerKelvin
	UnitVoltPerMetre
	UnitWeber
	UnitCandela
	UnitCandelaPerSquareMetre
	UnitKelvinPerHour
	UnitKelvinPerMinute
	UnitJouleSecond
	UnitRadianPerSecond
	UnitSquareMetrePerNewton
	UnitKilogramPerCubicMetre
	UnitNewtonSecond
	UnitNewtonPerMetre
	UnitWattPerMetreKelvin
	UnitMicrosiemens
	UnitCubicFootPerHour
	UnitUSGallonsPerHour
	UnitKilometre
	UnitMicrometreMicron
	UnitGram
	UnitMilligram
	UnitMillilitre
	UnitMillilitrePerSecond
	UnitDecibel
	UnitDecibelsMillivolt
	UnitDecibelsVolt
	UnitMillisiemens
	UnitWattHoursReactive
	UnitKilowattHoursReactive
	UnitMegawattHoursReactive
	UnitMillimetersOfWater
	UnitPerMille
	UnitGramsPerGram
	UnitKilogramPerKilogram
	UnitGramsPerKilogram
	UnitMilligramPerGram
	UnitMilligramPerKilogram
	UnitGramPerMillilitre
	UnitGramPerLitre
	UnitMilligramPerLitre
	UnitMicrogramPerLitre
	UnitGramPerCubicMetre
	UnitMilligramPerCubicMetre
	UnitMicrogramPerCubicMetre
	UnitNanogramsPerCubicMeter
	UnitGramPerCubicCentimetre
	UnitBecquerel
	UnitKilobecquerel
	UnitMegabecquerel
	UnitGray
	UnitMilligray
	UnitMicrogray
	UnitSievert
	UnitMillisievert
	UnitMicrosieverts
	UnitMicrosievertPerHour
	UnitDecibelsA
	UnitNephelometricTurbidityUnit
	UnitPhPotentialOfHydrogen
	UnitGramPerSquareMetre
	UnitMinutesPerDegreeKelvin
)

var UnitMap = map[uint16]string{
	UnitSquareMetre: "Square Metre",
	UnitSquareFoot: "Square Foot",
	UnitMilliampere: "Milliampere",
	UnitAmpere: "Ampere",
	UnitOhm: "Ohm",
	UnitVolt: "Volt",
	UnitKilovolt: "Kilovolt",
	UnitMegavolt: "Megavolt",
	UnitVoltAmpere: "Volt-Ampere",
	UnitKilovoltAmpere: "Kilovolt-Ampere",
	UnitMegavoltAmpere: "Megavolt-Ampere",
	UnitVoltAmperesReactive: "Volt-Amperes Reactive",
	UnitKilovoltAmpereReactive: "Kilovolt-Ampere Reactive",
	UnitMegavoltAmperesReactive: "Megavolt-Amperes Reactive",
	UnitDegreesPhase: "Degrees Phase",
	UnitPowerFactor: "Power Factor",
	UnitJoule: "Joule",
	UnitKilojoule: "Kilojoule",
	UnitWattHour: "Watt Hour",
	UnitKilowattHour: "Kilowatt Hour",
	UnitBritishThermalUnitMean: "British Thermal Unit (Mean)",
	UnitThermUS: "Therm (US)",
	UnitTonUSPerHour: "Ton (US) per Hour",
	UnitJoulesPerKilogramDryAir: "Joules per Kilogram Dry Air",
	UnitBtusPerPoundDryAir: "BTUs per Pound Dry Air",
	UnitCyclesPerHour: "Cycles per Hour",
	UnitCyclesPerMinute: "Cycles per Minute",
	UnitHertz: "Hertz",
	UnitGramsOfWaterPerKilogramDryAir: "Grams of Water per Kilogram Dry Air",
	UnitPercentRelativeHumidity: "Percent Relative Humidity",
	UnitMillimetre: "Millimetre",
	UnitMetre: "Metre",
	UnitInch: "Inch",
	UnitFoot: "Foot",
	UnitWattsPerSquareFoot: "Watts per Square Foot",
	UnitWattsPerSquareMeter: "Watts per Square Meter",
	UnitLumen: "Lumen",
	UnitLux: "Lux",
	UnitFootcandle: "Footcandle",
	UnitKilogram: "Kilogram",
	UnitPoundsMass: "Pounds Mass",
	UnitTons: "Tons",
	UnitKilogramPerSecond: "Kilogram per Second",
	UnitKilogramPerMinute: "Kilogram per Minute",
	UnitKilogramPerHour: "Kilogram per Hour",
	UnitPoundsMassPerMinute: "Pounds Mass per Minute",
	UnitPoundsMassPerHour: "Pounds Mass per Hour",
	UnitWatt: "Watt",
	UnitKilowatt: "Kilowatt",
	UnitMegawatt: "Megawatt",
	UnitBritishThermalUnitThermochemicalPerHour: "British Thermal Unit (Thermochemical) per Hour",
	UnitHorsepower: "Horsepower",
	UnitTonsRefrigeration: "Tons Refrigeration",
	UnitPascal: "Pascal",
	UnitKilopascal: "Kilopascal",
	UnitBarUnitOfPressure: "Bar (Unit of Pressure)",
	UnitPoundForcePerSquareInch: "Pound-Force per Square Inch",
	UnitCentimetersOfWater: "Centimeters of Water",
	UnitInchOfWater: "Inch of Water",
	UnitMillimetersOfMercury: "Millimeters of Mercury",
	UnitCentimetreOfMercury: "Centimetre of Mercury",
	UnitInchOfMercury: "Inch of Mercury",
	UnitDegreeCelsius: "Degree Celsius",
	UnitKelvin: "Kelvin",
	UnitDegreeFahrenheit: "Degree Fahrenheit",
	UnitDegreeDaysCelsius: "Degree Days Celsius",
	UnitDegreeDaysFahrenheit: "Degree Days Fahrenheit",
	UnitYears: "Years",
	UnitMonth: "Month",
	UnitWeek: "Week",
	UnitDay: "Day",
	UnitHour: "Hour",
	UnitMinuteUnitOfTime: "Minute (Unit of Time)",
	UnitSecondUnitOfTime: "Second (Unit of Time)",
	UnitMetrePerSecond: "Metre per Second",
	UnitKilometrePerHour: "Kilometre per Hour",
	UnitFootPerSecond: "Foot per Second",
	UnitFootPerMinute: "Foot per Minute",
	UnitMilePerHourStatuteMile: "Mile per Hour (Statute Mile)",
	UnitCubicFoot: "Cubic Foot",
	UnitCubicMetre: "Cubic Metre",
	UnitImperialGallons: "Imperial Gallons",
	UnitLitre: "Litre",
	UnitGallonUS: "Gallon (US)",
	UnitCubicFootPerMinute: "Cubic Foot per Minute",
	UnitCubicMetrePerSecond: "Cubic Metre per Second",
	UnitImperialGallonPerMinute: "Imperial Gallon per Minute",
	UnitLitrePerSecond: "Litre per Second",
	UnitLitrePerMinute: "Litre per Minute",
	UnitUSGallonPerMinute: "US Gallon per Minute",
	UnitDegreeUnitOfAngle: "Degree (Unit of Angle)",
	UnitDegreeCelsiusPerHour: "Degree Celsius per Hour",
	UnitDegreeCelsiusPerMinute: "Degree Celsius per Minute",
	UnitDegreeFahrenheitPerHour: "Degree Fahrenheit per Hour",
	UnitDegreeFahrenheitPerMinute: "Degree Fahrenheit per Minute",
	UnitNoUnits: "No Units",
	UnitPartsPerMillion: "Parts per Million",
	UnitPartsPerBillion: "Parts per Billion",
	UnitPercent: "Percent",
	UnitPercentPerSecond: "Percent per Second",
	UnitPerMinute: "Per Minute",
	UnitPerSecond: "Per Second",
	UnitPsiPerDegreeFahrenheit: "PSI per Degree Fahrenheit",
	UnitRadians: "Radians",
	UnitRevolutionsPerMinute: "Revolutions per Minute",
	UnitCurrency1: "Currency 1",
	UnitCurrency2: "Currency 2",
	UnitCurrency3: "Currency 3",
	UnitCurrency4: "Currency 4",
	UnitCurrency5: "Currency 5",
	UnitCurrency6: "Currency 6",
	UnitCurrency7: "Currency 7",
	UnitCurrency8: "Currency 8",
	UnitCurrency9: "Currency 9",
	UnitCurrency10: "Currency 10",
	UnitSquareInch: "Square Inch",
	UnitSquareCentimetre: "Square Centimetre",
	UnitBritishThermalUnitInternationalTablePerPound: "British Thermal Unit (International Table) per Pound",
	UnitCentimetre: "Centimetre",
	UnitPoundsMassPerSecond: "Pounds Mass per Second",
	UnitDeltaDegreesFahrenheit: "Delta Degrees Fahrenheit",
	UnitDeltaDegreesKelvin: "Delta Degrees Kelvin",
	UnitKiloohm: "Kiloohm",
	UnitMegaohm: "Megaohm",
	UnitMillivolt: "Millivolt",
	UnitKilojoulePerKilogram: "Kilojoule per Kilogram",
	UnitMegajoule: "Megajoule",
	UnitJoulesPerDegreeKelvin: "Joules per Degree Kelvin",
	UnitJoulesPerKilogramDegreeKelvin: "Joules per Kilogram Degree Kelvin",
	UnitKilohertz: "Kilohertz",
	UnitMegahertz: "Megahertz",
	UnitPerHour: "Per Hour",
	UnitMilliwatt: "Milliwatt",
	UnitHectopascal: "Hectopascal",
	UnitMillibar: "Millibar",
	UnitCubicMetrePerHour: "Cubic Metre per Hour",
	UnitLitersPerHour: "Liters per Hour",
	UnitKilowattHoursPerSquareMeter: "Kilowatt Hours per Square Meter",
	UnitKilowattHoursPerSquareFoot: "Kilowatt Hours per Square Foot",
	UnitMegajoulesPerSquareMeter: "Megajoules per Square Meter",
	UnitMegajoulesPerSquareFoot: "Megajoules per Square Foot",
	UnitWattsPerSquareMeterDegreeKelvin: "Watts per Square Meter Degree Kelvin",
	UnitCubicFeetPerSecond: "Cubic Feet per Second",
	UnitPercentObscurationPerFoot: "Percent Obscuration per Foot",
	UnitPercentObscurationPerMeter: "Percent Obscuration per Meter",
	UnitMilliohm: "Milliohm",
	UnitMegawattHour1000KWh: "Megawatt Hour (1000 KWh)",
	UnitKiloBtus: "Kilo BTUs",
	UnitMegaBtus: "Mega BTUs",
	UnitKilojoulesPerKilogramDryAir: "Kilojoules per Kilogram Dry Air",
	UnitMegajoulesPerKilogramDryAir: "Megajoules per Kilogram Dry Air",
	UnitKilojoulesPerDegreeKelvin: "Kilojoules per Degree Kelvin",
	UnitMegajoulesPerDegreeKelvin: "Megajoules per Degree Kelvin",
	UnitNewton: "Newton",
	UnitGramPerSecond: "Gram per Second",
	UnitGramPerMinute: "Gram per Minute",
	UnitTonnePerHour: "Tonne per Hour",
	UnitKiloBtusPerHour: "Kilo BTUs per Hour",
	UnitHundredthsSeconds: "Hundredths of Seconds",
	UnitMillisecond: "Millisecond",
	UnitNewtonMetre: "Newton Metre",
	UnitMillimetrePerSecond: "Millimetre per Second",
	UnitMillimetrePerMinute: "Millimetre per Minute",
	UnitMetrePerMinute: "Metre per Minute",
	UnitMetrePerHour: "Metre per Hour",
	UnitCubicMetrePerMinute: "Cubic Metre per Minute",
	UnitMetrePerSecondSquared: "Metre per Second Squared",
	UnitAmperePerMetre: "Ampere per Metre",
	UnitAmperePerSquareMetre: "Ampere per Square Metre",
	UnitAmpereSquareMetre: "Ampere Square Metre",
	UnitFarad: "Farad",
	UnitHenry: "Henry",
	UnitOhmMetre: "Ohm Metre",
	UnitSiemens: "Siemens",
	UnitSiemensPerMetre: "Siemens per Metre",
	UnitTesla: "Tesla",
	UnitVoltPerKelvin: "Volt per Kelvin",
	UnitVoltPerMetre: "Volt per Metre",
	UnitWeber: "Weber",
	UnitCandela: "Candela",
	UnitCandelaPerSquareMetre: "Candela per Square Metre",
	UnitKelvinPerHour: "Kelvin per Hour",
	UnitKelvinPerMinute: "Kelvin per Minute",
	UnitJouleSecond: "Joule Second",
	UnitRadianPerSecond: "Radian per Second",
	UnitSquareMetrePerNewton: "Square Metre per Newton",
	UnitKilogramPerCubicMetre: "Kilogram per Cubic Metre",
	UnitNewtonSecond: "Newton Second",
	UnitNewtonPerMetre: "Newton per Metre",
	UnitWattPerMetreKelvin: "Watt per Metre Kelvin",
	UnitMicrosiemens: "Microsiemens",
	UnitCubicFootPerHour: "Cubic Foot per Hour",
	UnitUSGallonsPerHour: "US Gallons per Hour",
	UnitKilometre: "Kilometre",
	UnitMicrometreMicron: "Micrometre (Micron)",
	UnitGram: "Gram",
	UnitMilligram: "Milligram",
	UnitMillilitre: "Millilitre",
	UnitMillilitrePerSecond: "Millilitre per Second",
	UnitDecibel: "Decibel",
	UnitDecibelsMillivolt: "Decibels Millivolt",
	UnitDecibelsVolt: "Decibels Volt",
	UnitMillisiemens: "Millisiemens",
	UnitWattHoursReactive: "Watt Hours Reactive",
	UnitKilowattHoursReactive: "Kilowatt Hours Reactive",
	UnitMegawattHoursReactive: "Megawatt Hours Reactive",
	UnitMillimetersOfWater: "Millimeters of Water",
	UnitPerMille: "Per Mille",
	UnitGramsPerGram: "Grams per Gram",
	UnitKilogramPerKilogram: "Kilogram per Kilogram",
	UnitGramsPerKilogram: "Grams per Kilogram",
	UnitMilligramPerGram: "Milligram per Gram",
	UnitMilligramPerKilogram: "Milligram per Kilogram",
	UnitGramPerMillilitre: "Gram per Millilitre",
	UnitGramPerLitre: "Gram per Litre",
	UnitMilligramPerLitre: "Milligram per Litre",
	UnitMicrogramPerLitre: "Microgram per Litre",
	UnitGramPerCubicMetre: "Gram per Cubic Metre",
	UnitMilligramPerCubicMetre: "Milligram per Cubic Metre",
	UnitMicrogramPerCubicMetre: "Microgram per Cubic Metre",
	UnitNanogramsPerCubicMeter: "Nanograms per Cubic Meter",
	UnitGramPerCubicCentimetre: "Gram per Cubic Centimetre",
	UnitBecquerel: "Becquerel",
	UnitKilobecquerel: "Kilobecquerel",
	UnitMegabecquerel: "Megabecquerel",
	UnitGray: "Gray",
	UnitMilligray: "Milligray",
	UnitMicrogray: "Microgray",
	UnitSievert: "Sievert",
	UnitMillisievert: "Millisievert",
	UnitMicrosieverts: "Microsieverts",
	UnitMicrosievertPerHour: "Microsievert per Hour",
	UnitDecibelsA: "Decibels (A)",
	UnitNephelometricTurbidityUnit: "Nephelometric Turbidity Unit",
	UnitPhPotentialOfHydrogen: "pH (Potential of Hydrogen)",
	UnitGramPerSquareMetre: "Gram per Square Metre",
	UnitMinutesPerDegreeKelvin: "Minutes per Degree Kelvin",
}
