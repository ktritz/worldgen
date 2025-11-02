#!/usr/bin/env python3
"""
Test script to verify the updated UI works with the modularized backend
"""
import requests
import json

def test_icosphere_generation():
    """Test icosphere generation first"""
    print("Testing icosphere generation...")
    url = "http://localhost:8080/api/generate_icosphere"
    
    payload = {
        "subdivision": 2,
        "seed": 12345
    }
    
    try:
        response = requests.post(url, json=payload, timeout=30)
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Icosphere generation successful")
            print(f"   - Vertices: {len(data.get('icosphereData', {}).get('vertices', [])) // 3}")
            print(f"   - Faces: {len(data.get('icosphereData', {}).get('faces', [])) // 3}")
            return data
        else:
            print(f"❌ Icosphere generation failed: {response.status_code}")
            print(f"   Response: {response.text}")
            return None
    except Exception as e:
        print(f"❌ Icosphere generation error: {e}")
        return None

def test_landgen_with_updated_params(icosphere_data):
    """Test land generation with the new parameter structure"""
    print("\nTesting land generation with updated parameters...")
    url = "http://localhost:8080/api/generate_land"
    
    payload = {
        # General land parameters
        "landSeed": 12345,
        
        # Updated tectonic parameters
        "numPlates": 8,
        "baseSpeed": 0.01,
        "speedVariationFactor": 0.5,
        "targetContinentalProportion": 0.35,
        "numInitialContinentalSeeds": 3,
        
        # Elevation parameters
        "noiseScale": 1.0,
        "noiseOctaves": 4,
        "noisePersistence": 0.5,
        "noiseLacunarity": 2.0,
        "elevationMultiplier": 100,
        
        # New tectonic boundary effects
        "characteristicFalloffDistance": 0.15,
        "maxBoundaryEffectDistance": 0.45,
        "convergentBoundaryStrength": 1000,
        "divergentBoundaryStrength": 500,
        
        # Base mesh data
        "baseIcosphereData": icosphere_data.get("icosphereData"),
        "baseVoronoiData": icosphere_data.get("voronoiData"),
        "icosphereSubdivisions": icosphere_data.get("subdivisionLevel", 2),
        "landOutputName": "test_output.png"
    }
    
    try:
        response = requests.post(url, json=payload, timeout=60)
        if response.status_code == 200:
            data = response.json()
            print(f"✅ Land generation successful")
            print(f"   - Status: {data.get('status')}")
            
            # Check elevation data
            elevation_data = data.get('elevationData', {})
            cell_elevations = elevation_data.get('cellElevations', {})
            if cell_elevations:
                elevations = list(cell_elevations.values())
                print(f"   - Elevation points: {len(elevations)}")
                print(f"   - Elevation range: {min(elevations):.1f} to {max(elevations):.1f} meters")
            else:
                print("   - No elevation data found")
            
            return data
        else:
            print(f"❌ Land generation failed: {response.status_code}")
            print(f"   Response: {response.text}")
            return None
    except Exception as e:
        print(f"❌ Land generation error: {e}")
        return None

def main():
    print("🧪 Testing UI Integration with Modularized Backend")
    print("=" * 50)
    
    # Test icosphere generation first
    icosphere_data = test_icosphere_generation()
    if not icosphere_data:
        print("❌ Cannot continue without icosphere data")
        return
    
    # Test land generation with new parameters
    landgen_data = test_landgen_with_updated_params(icosphere_data)
    if landgen_data:
        print("\n✅ All tests passed! UI integration successful.")
    else:
        print("\n❌ Land generation test failed")

if __name__ == "__main__":
    main()