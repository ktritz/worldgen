#!/usr/bin/env python3
"""
Simple test data generation using Python to avoid Go struct issues
"""
import requests
import json
import os

def main():
    print("🧪 Generating Test Data for Visual Analysis")
    print("==========================================")

    # Step 1: Generate Icosphere
    print("1. Generating icosphere and Voronoi data...")
    
    icosphere_payload = {
        "subdivision": 3,  # Higher subdivision for more detail
        "seed": 42
    }

    try:
        response = requests.post("http://localhost:8080/api/generate", json=icosphere_payload, timeout=30)
        if response.status_code != 200:
            print(f"❌ Icosphere generation failed: {response.status_code}")
            print(f"Response: {response.text}")
            return
        
        icosphere_data = response.json()
        print("✅ Icosphere generated successfully")
        
        # Save icosphere data for inspection
        with open("icosphere_data.json", "w") as f:
            json.dump(icosphere_data, f, indent=2)
        print("💾 Saved: icosphere_data.json")

    except Exception as e:
        print(f"❌ Failed to generate icosphere: {e}")
        return

    # Step 2: Generate Land with updated parameters
    print("2. Generating land with tectonic plates and elevation...")

    land_payload = {
        # General parameters
        "landSeed": 42,

        # Tectonic parameters (updated field names)
        "numPlates": 12,
        "baseSpeed": 0.02,
        "speedVariationFactor": 0.8,
        "targetContinentalProportion": 0.4,
        "numInitialContinentalSeeds": 4,

        # Elevation parameters  
        "noiseScale": 1.0,
        "noiseOctaves": 4,
        "noisePersistence": 0.5,
        "noiseLacunarity": 2.0,
        "elevationMultiplier": 150,

        # Tectonic boundary effects
        "characteristicFalloffDistance": 0.1,
        "maxBoundaryEffectDistance": 0.3,
        "convergentBoundaryStrength": 2000,
        "divergentBoundaryStrength": 800,

        # Mesh data from icosphere generation
        "baseIcosphereData": icosphere_data["icosphereData"],
        "baseVoronoiData": icosphere_data["voronoiData"],
        "icosphereSubdivisions": icosphere_data["subdivisionLevel"],
        "landOutputName": "test_analysis_output.png"
    }

    try:
        response = requests.post("http://localhost:8080/api/generate_land", json=land_payload, timeout=60)
        if response.status_code != 200:
            print(f"❌ Land generation failed: {response.status_code}")
            print(f"Response: {response.text}")
            return
        
        land_data = response.json()
        print("✅ Land generated successfully")
        
        # Save land data for inspection
        with open("land_data.json", "w") as f:
            json.dump(land_data, f, indent=2)
        print("💾 Saved: land_data.json")

        # Generate analysis
        generate_analysis(icosphere_data, land_data)

    except Exception as e:
        print(f"❌ Failed to generate land: {e}")
        return

    print("\n🎯 Test data generation complete!")
    print("📊 Check the following files for visual analysis:")
    print("   - icosphere_data.json (icosphere and Voronoi data)")
    print("   - land_data.json (land generation results)")
    print("   - analysis_summary.txt (statistical summary)")

def generate_analysis(icosphere_data, land_data):
    """Generate analysis summary"""
    summary = []
    summary.append("LANDGEN TEST DATA ANALYSIS SUMMARY")
    summary.append("===================================\n")

    # Icosphere analysis
    ico_data = icosphere_data.get("icosphereData", {})
    vertices = ico_data.get("vertices", [])
    faces = ico_data.get("faces", [])
    
    summary.append(f"Icosphere Vertices: {len(vertices)//3}")
    summary.append(f"Icosphere Faces: {len(faces)//3}")

    vor_data = icosphere_data.get("voronoiData", {})
    cells = vor_data.get("cells", [])
    summary.append(f"Voronoi Cells (Potential Plates): {len(cells)}\n")

    # Elevation analysis
    elev_data = land_data.get("elevationData", {})
    cell_elevations = elev_data.get("cellElevations", {})
    
    if cell_elevations:
        elevations = list(cell_elevations.values())
        
        if elevations:
            min_elev = min(elevations)
            max_elev = max(elevations)
            avg_elev = sum(elevations) / len(elevations)
            
            summary.append("Elevation Analysis:")
            summary.append(f"  Sites with elevation: {len(elevations)}")
            summary.append(f"  Elevation range: {min_elev:.1f} to {max_elev:.1f} meters")
            summary.append(f"  Average elevation: {avg_elev:.1f} meters")
            summary.append(f"  Total relief: {max_elev - min_elev:.1f} meters")

            # Categorize elevations
            oceanic = sum(1 for e in elevations if e < 0)
            lowland = sum(1 for e in elevations if 0 <= e < 200)
            highland = sum(1 for e in elevations if 200 <= e < 1000)
            mountain = sum(1 for e in elevations if e >= 1000)
            
            total = len(elevations)
            summary.append(f"\nElevation Distribution:")
            summary.append(f"  Oceanic (< 0m): {oceanic} ({oceanic/total*100:.1f}%)")
            summary.append(f"  Lowland (0-200m): {lowland} ({lowland/total*100:.1f}%)")
            summary.append(f"  Highland (200-1000m): {highland} ({highland/total*100:.1f}%)")
            summary.append(f"  Mountain (>1000m): {mountain} ({mountain/total*100:.1f}%)")

    summary.append(f"\nExpected Behavior:")
    summary.append("- Should see mix of oceanic (negative) and continental (positive) elevations")
    summary.append("- Continental plates should show higher base elevations (200-600m range)")
    summary.append("- Oceanic plates should show lower base elevations (-4500 to -3500m range)")
    summary.append("- Convergent boundaries should create mountains (elevated areas)")
    summary.append("- Divergent boundaries should create rifts (depressed areas)")
    summary.append("- Total relief should be significant (>3000m) indicating tectonic effects")

    with open("analysis_summary.txt", "w") as f:
        f.write("\n".join(summary))
    
    print("📊 Saved: analysis_summary.txt")

if __name__ == "__main__":
    main()