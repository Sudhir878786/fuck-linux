// Arduino Accelerometer Sketch for spank
// ----------------------------------------
// Reads an accelerometer (e.g., ADXL345, MPU6050, LIS3DH) and sends
// newline-separated X,Y,Z float values over Serial at 115200 baud.
//
// Wire your accelerometer to the Arduino via I2C or SPI,
// then connect the Arduino to the Linux machine via USB.
// The serial device will typically appear as /dev/ttyUSB0 or /dev/ttyACM0.
//
// Expected output format (one line per reading):
//   0.12,-0.03,9.81
//   0.15,-0.01,9.78
//   3.42,1.20,14.50   <-- spike = slap detected
//
// This example uses the Adafruit LIS3DH library.
// Install via Arduino IDE: Sketch > Include Library > Manage Libraries > "Adafruit LIS3DH"

#include <Wire.h>
#include <Adafruit_LIS3DH.h>
#include <Adafruit_Sensor.h>

Adafruit_LIS3DH lis = Adafruit_LIS3DH();

void setup() {
  Serial.begin(115200);
  while (!Serial) delay(10);

  if (!lis.begin(0x18)) {  // Default I2C address
    Serial.println("# ERROR: LIS3DH not found");
    while (1) delay(100);
  }

  // Set range: LIS3DH_RANGE_2_G, _4_G, _8_G, _16_G
  lis.setRange(LIS3DH_RANGE_4_G);

  // Set data rate: LIS3DH_DATARATE_100_HZ is a good balance
  lis.setDataRate(LIS3DH_DATARATE_100_HZ);

  Serial.println("# LIS3DH ready");
}

void loop() {
  lis.read();

  // Convert raw values to m/s² (LIS3DH: 1g ≈ 9.81 m/s²)
  sensors_event_t event;
  lis.getEvent(&event);

  // Output format: X,Y,Z in m/s²
  Serial.print(event.acceleration.x, 4);
  Serial.print(",");
  Serial.print(event.acceleration.y, 4);
  Serial.print(",");
  Serial.println(event.acceleration.z, 4);

  delay(10);  // ~100 Hz output rate
}
