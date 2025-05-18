<!-----



Conversion time: 2.207 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β44
* Sat May 17 2025 18:07:39 GMT-0700 (PDT)
* Source doc: Tectonic plate generation
* Tables are currently converted to HTML tables.
----->



# A Geologically-Informed Model for Procedural Tectonic Plate Generation on an Icosphere


## 1. Simulating Tectonic Plates for Procedural Worlds: An Overview


### Purpose

This report outlines a methodology for modeling tectonic plates on an icosphere using Voronoi cells, aimed at the procedural generation of realistic, large-scale geographical features. The core objective is to develop efficient models that closely emulate real-world geological characteristics, catering to the needs of procedural world generation projects that require a balance of scientific plausibility and computational tractability.


### Balancing Realism and Efficiency

The simulation of complex geological phenomena, such as plate tectonics, presents an inherent challenge in balancing scientific fidelity with the computational constraints of procedural generation. A full-scale simulation of Earth's geodynamics is well beyond the scope of typical procedural content generation. Therefore, this report focuses on simplified, yet scientifically-grounded, models. These models abstract intricate physical details where necessary, ensuring that the core behaviors and visual characteristics of tectonic plates are preserved without imposing an undue computational burden. This approach is designed to provide models that are both realistic enough for immersive world generation and efficient enough for practical implementation.

The choice of an icosphere as the base geometry, combined with Voronoi cells generated from Centroidal Voronoi Tessellation (CVT) centroids, provides a robust and common starting point for partitioning a sphere into initial plate-like regions.<sup>1</sup> An icosphere offers a relatively uniform discretization of a sphere, minimizing the distortions often associated with latitude-longitude grids. Voronoi diagrams naturally partition space based on proximity to a set of seed points (the CVT centroids in this context).<sup>4</sup> CVTs, achieved through iterative methods like Lloyd's algorithm, further refine this by ensuring each seed point is also the geometric center of its Voronoi cell, leading to more regular and evenly distributed initial polygons.<sup>5</sup> This framework is well-established in procedural generation for spherical worlds <sup>3</sup> and forms the foundation upon which the subsequent geological characteristics will be built. The primary task then becomes imbuing these initial, somewhat uniform, Voronoi cells with the distinct and varied characteristics of real tectonic plates.


### Structure of the Report

This document will systematically guide the reader through the necessary concepts and procedures:



1. **Core Principles of Plate Tectonics:** A review of the fundamental geological mechanisms and definitions relevant to procedural modeling.
2. **Oceanic vs. Continental Plates:** Detailing the fundamental distinctions between these plate types and their impact on geological features.
3. **Initial Plate Generation:** Describing the process of creating initial plate structures on the icosphere using Voronoi tessellation.
4. **Refining Voronoi Plate Boundaries:** Presenting techniques to transform geometrically simple Voronoi edges into more geologically realistic plate boundaries.
5. **Modeling Average Plate Elevation:** Discussing methods to assign plausible average elevations to different plate types.
6. **Pseudo-code Algorithms:** Providing simplified algorithms for implementing the proposed models.
7. **Conclusion and Further Considerations:** Summarizing the approach and suggesting avenues for future development.


## 2. Core Principles of Plate Tectonics for Procedural Modeling

Understanding the fundamental principles of plate tectonics is crucial for developing a realistic procedural model. This section outlines the key concepts that will inform the generation algorithms.


### 2.1. Driving Mechanisms: A Simplified View

The Earth's internal heat, originating from the radioactive decay of elements within its interior and residual heat from the planet's formation, serves as the ultimate energy source for plate tectonics.<sup>11</sup> This thermal energy drives a planetary-scale convection system.

While early theories posited that mantle convection cells directly drag the lithospheric plates along their surface <sup>11</sup>, current dynamic models emphasize the role of gravity-driven forces acting on the plates themselves. The two primary forces are:



* **Ridge Push:** This gravitational force arises at mid-ocean ridges, which are elevated due to the hot, buoyant mantle material beneath them. The newly formed, hot, and therefore less dense lithosphere at the ridge crest effectively slides "downhill" away from the ridge axis under the influence of gravity.<sup>11</sup>
* **Slab Pull:** This is considered a dominant, if not the most significant, driving force. As old, cold, and therefore dense oceanic lithosphere collides with another plate, it sinks (subducts) into the more ductile asthenosphere. The immense weight of this sinking slab pulls the rest of the plate along with it.<sup>11</sup>

Mantle convection remains an integral part of this system. It influences the temperature and viscosity of the asthenosphere, contributes to the upwelling of material at ridges, and is linked to the downwelling at subduction zones. However, it is now understood that plates are not merely passive passengers on conveyor-belt-like convection cells; rather, they are active participants in a gravity-driven system.<sup>11</sup>

For the purpose of procedural generation, a full simulation of mantle fluid dynamics is computationally infeasible. However, the concepts of ridge push and slab pull can be abstracted to inform plate motion rules. For example, plates with significant spreading ridges might be assigned a velocity component directed away from these ridges, while plates with substantial subducting edges could experience an increased tendency to move towards the subduction zone, potentially influencing their assigned velocities.


### 2.2. Plate Definition: Lithosphere, Crust, Asthenosphere

Tectonic plates are defined as large, rigid segments of the **lithosphere**. The lithosphere is the Earth's cool, strong, outermost mechanical layer and is composed of two parts: the **crust** (which can be either oceanic or continental in nature) and the **uppermost, rigid portion of the mantle**.<sup>11</sup> The thickness of the lithosphere varies. Oceanic lithosphere is typically thinner, ranging from about 6 km at mid-ocean ridges where it forms, to over 100 km in older regions approaching subduction zones.<sup>15</sup> Continental lithosphere is generally thicker, averaging around 200 km, though this can vary considerably between different geological settings like basins, mountain ranges, and stable cratonic interiors.<sup>15</sup>

These rigid lithospheric plates "float" and move over the **asthenosphere**. The asthenosphere is a mechanically weaker, hotter, and more ductile (capable of slow, viscous flow over geological timescales) layer within the upper mantle, situated directly beneath the lithosphere.<sup>12</sup> The ability of the asthenosphere to deform allows the overlying rigid lithospheric plates to move across the Earth's surface. The distinction between the rigid lithosphere and the flowing asthenosphere is based on their mechanical properties and heat transfer mechanisms rather than solely on chemical composition.<sup>15</sup>

In the context of the procedural model, each Voronoi cell will represent one of these rigid lithospheric plates. The surface of the icosphere corresponds to the surface of these plates. The asthenosphere itself is not explicitly modeled as a distinct layer but serves as the conceptual "slippery surface" upon which the plates translate and rotate.


### 2.3. Plate Kinematics: Motion on a Sphere

The movement of tectonic plates across the Earth's spherical surface is described by specific kinematic principles.


#### 2.3.1. Plate Velocities

Tectonic plates move at a range of speeds. Typical relative velocities between plates range from 0 to 10 centimeters per year (cm/yr).<sup>15</sup> Some plates, like the Nazca plate, can move faster, reaching speeds of up to approximately 160 millimeters per year (16 cm/yr).<sup>15</sup> Slower spreading rates, such as those at the Mid-Atlantic Ridge, are around 10 to 40 mm/yr (1-4 cm/yr).<sup>15</sup> These rates are often compared to the speed at which human fingernails or hair grow, providing a relatable sense of their pace.<sup>15</sup> The USGS notes that average rates of motion range from less than 1 to more than 15 cm/yr.<sup>17</sup> For procedural generation, assigning random velocities to plates within this observed geological range (e.g., 1-16 cm/yr) will contribute to a realistic depiction of plate dynamics.


#### 2.3.2. Euler Poles and Rotation

The motion of a rigid body, such as a tectonic plate, on the surface of a sphere is described by **Euler's fixed point theorem**. This theorem states that any displacement of a rigid body on a sphere's surface can be represented as a single rotation about an appropriately chosen axis that passes through the center of the sphere.<sup>21</sup> The points where this axis intersects the sphere's surface are known as the **Euler poles** for that specific rotation.

Each tectonic plate has an Euler pole (defined by a latitude and longitude) and an angular velocity (typically measured in degrees per million years, deg/Myr, or radians/Myr) that describes its motion relative to another plate or a fixed reference frame (such as a hotspot reference frame or a no-net-rotation frame).<sup>24</sup> The NUVEL-1A model, for example, is a global plate motion model that defines these Euler vectors for Earth's major plates.<sup>24</sup>

The linear velocity (V) at any point on a plate's surface can be calculated from its angular velocity (ω) and its angular distance (θ) from the plate's Euler pole. The relationship is given by V=ω⋅Rearth​⋅sin(θ), where Rearth​ is the radius of the Earth.<sup>24</sup> The velocity is zero at the Euler pole itself and reaches its maximum at an angular distance of 90 degrees from the pole (i.e., along the "equator" relative to that pole). The direction of motion at any point is tangential to the small circle centered on the Euler pole that passes through that point.

For the procedural model, assigning a random Euler pole (a point on the icosphere) and a random angular velocity (within geologically plausible ranges, e.g., derived from models like NUVEL-1A) to each generated plate will define its complete motion. This allows for the calculation of a specific velocity vector (magnitude and direction) for any location on that plate, which is fundamental for simulating plate interactions, drift, and the resulting geological features.


<table>
  <tr>
   <td><strong>Plate Name</strong>
   </td>
   <td><strong>Typical Linear Velocity Range (cm/year)</strong>
   </td>
   <td><strong>Example Euler Pole (Lat °N, Lon °E, Ang. Vel. deg/Myr)</strong>
   </td>
  </tr>
  <tr>
   <td>Pacific
   </td>
   <td>5 - 10+ (locally up to 16 for Nazca)
   </td>
   <td>~ -62, ~100, ~0.8-1.0
   </td>
  </tr>
  <tr>
   <td>North American
   </td>
   <td>1 - 4
   </td>
   <td>~ 49, ~-78, ~0.2-0.3
   </td>
  </tr>
  <tr>
   <td>Eurasian
   </td>
   <td>0.5 - 3
   </td>
   <td>~ 61, ~-86, ~0.1-0.25
   </td>
  </tr>
  <tr>
   <td>African
   </td>
   <td>1 - 3
   </td>
   <td>~ 59, ~-73, ~0.2-0.3
   </td>
  </tr>
  <tr>
   <td>Antarctic
   </td>
   <td>0.5 - 2
   </td>
   <td>~ 64, ~-84, ~0.2-0.25
   </td>
  </tr>
  <tr>
   <td>Indo-Australian
   </td>
   <td>5 - 8
   </td>
   <td>~ 60 (Aus), ~-30 (India), ~1.1-1.2
   </td>
  </tr>
  <tr>
   <td>South American
   </td>
   <td>1 - 4
   </td>
   <td>~ 55, ~-86, ~0.15-0.25
   </td>
  </tr>
  <tr>
   <td>Nazca
   </td>
   <td>6 - 16
   </td>
   <td>~ 56, ~-90, ~1.0-1.4
   </td>
  </tr>
</table>


*Note: Euler pole values are highly generalized examples inspired by models like NUVEL-1A <sup>25</sup> for illustrative purposes and may not represent precise current estimates for specific plate pairs. Angular velocities are also generalized. The key is the concept and the range.*


### 2.4. Plate Size and Shape Characteristics


#### 2.4.1. Plate Size Distribution

The Earth's lithosphere is not broken into uniformly sized pieces. Instead, it is fragmented into a hierarchy of plates, typically described as having about a dozen major plates and numerous smaller ones.<sup>17</sup> More detailed geological models, such as that by Bird (2003), identify as many as 52 distinct lithospheric plates.<sup>26</sup> This indicates a wide range of plate areas, from vast oceanic plates like the Pacific Plate to much smaller microplates.

Statistical analyses of plate sizes suggest that their distribution is not random in a simple sense but may follow specific mathematical patterns. One prominent model, the "broken sheet" model, proposes that the diameters of individual plates are exponentially distributed.<sup>26</sup> This model arises from the concept of random linear subdivision of a surface, implying that if boundaries form through a somewhat random fracturing process, an exponential distribution of sizes would result. Other studies have suggested that plate sizes might follow power-law distributions or exhibit fractal sub-populations, indicating self-similar characteristics across different scales.<sup>26</sup> The table below, derived from data in <sup>29</sup>, illustrates the significant variation in plate areas.


<table>
  <tr>
   <td><strong>Tectonic Plate</strong>
   </td>
   <td><strong>Area (approx. million sq. km)</strong>
   </td>
   <td><strong>Category</strong>
   </td>
  </tr>
  <tr>
   <td>Pacific
   </td>
   <td>104.6
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>African
   </td>
   <td>58.5
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>Antarctic
   </td>
   <td>58.2
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>North American
   </td>
   <td>55.5
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>Eurasian
   </td>
   <td>48.6
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>Australian
   </td>
   <td>46.0
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>South American
   </td>
   <td>41.8
   </td>
   <td>Major
   </td>
  </tr>
  <tr>
   <td>Somali
   </td>
   <td>19.1
   </td>
   <td>Intermediate
   </td>
  </tr>
  <tr>
   <td>Nazca
   </td>
   <td>16.1
   </td>
   <td>Intermediate
   </td>
  </tr>
  <tr>
   <td>Philippine Sea
   </td>
   <td>5.5
   </td>
   <td>Minor
   </td>
  </tr>
  <tr>
   <td>Arabian
   </td>
   <td>5.0
   </td>
   <td>Minor
   </td>
  </tr>
  <tr>
   <td>Caribbean
   </td>
   <td>3.3
   </td>
   <td>Minor
   </td>
  </tr>
  <tr>
   <td>Cocos
   </td>
   <td>2.9
   </td>
   <td>Minor
   </td>
  </tr>
  <tr>
   <td>Scotia
   </td>
   <td>1.6
   </td>
   <td>Minor
   </td>
  </tr>
  <tr>
   <td>Juan de Fuca
   </td>
   <td>0.25
   </td>
   <td>Microplate
   </td>
  </tr>
  <tr>
   <td>North Andes
   </td>
   <td>0.97
   </td>
   <td>Minor (example)
   </td>
  </tr>
  <tr>
   <td>Conway Reef
   </td>
   <td>0.14
   </td>
   <td>Microplate
   </td>
  </tr>
</table>


*<sup>29</sup>*

An important geological observation is that smaller plates are not uniformly distributed across the globe but tend to be clustered along convergent plate boundaries, particularly in regions of complex tectonic interaction like the southwestern Pacific.<sup>26</sup> This clustering suggests that while initial large-scale fragmentation might have stochastic elements, the subsequent evolution and interaction of plates lead to further, more localized fragmentation in high-stress zones.

For procedural generation, these characteristics imply that simply generating Voronoi cells of roughly equal size will not be realistic. The number of initial CVT seeds should be chosen with this distribution in mind. One approach could be to generate a larger number of small initial Voronoi cells and then implement a merging algorithm based on a target statistical distribution (e.g., exponential or power-law for areas). Alternatively, one could start with a smaller number of seeds intended to represent major plates and then apply fragmentation rules, especially in procedurally generated convergent zones, to create smaller plates, mimicking the observed clustering. The "broken sheet" model, which posits that linear distances across plates are exponentially distributed, provides a mathematical basis for such size distributions.<sup>26</sup>

The initial random fracturing of the lithosphere, as suggested by models like the "broken sheet" theory <sup>26</sup>, provides a plausible starting point for plate formation. However, the Earth's tectonic system is dynamic and has evolved over billions of years. Processes such as the continuous creation of new crust at spreading ridges, the consumption of old crust at subduction zones, and the complex stresses and deformations at plate boundaries <sup>15</sup> actively modify plate sizes and shapes. The observed clustering of smaller plates near convergent boundaries <sup>26</sup> is a testament to these ongoing geological processes, where intense stress can lead to the fragmentation of larger plates or the formation of new, smaller microplates. This interplay between initial, possibly random, fragmentation and subsequent, geologically determined evolution means that a purely random Voronoi tessellation, even if adjusted for size distribution, will not fully capture the realistic arrangement of tectonic plates. While a full dynamic simulation of this evolution is beyond the scope of this report, the procedural model can incorporate elements that reflect these geological influences, for example, by biasing the "plate strength" heuristic (discussed in Section 5) or applying specific fragmentation algorithms in designated high-stress (e.g., convergent) zones during post-processing.


#### 2.4.2. Plate Shape - Beyond Simple Voronoi Polygons

The shapes of real tectonic plates are far removed from the simple, convex polygons typically generated by standard Voronoi tessellation. Plate boundaries exhibit a variety of complex geometries:



* **Arcuate (Curved) Shapes:** These are particularly characteristic of subduction zones, where the bending of the subducting plate often results in curved trenches and overlying volcanic island arcs.<sup>15</sup> The 3D geometry and curvature of these margins can significantly influence the stress regime and deformation patterns in the overriding plate.<sup>27</sup>
* **Segmented Shapes:** Mid-ocean ridges, where new oceanic crust is formed, are not continuous linear features. They are typically broken into numerous segments offset by transform faults. This results in a characteristic jagged, step-like boundary pattern.<sup>28</sup>
* **Relatively Linear Shapes:** Some long transform faults, which accommodate horizontal sliding between plates, can appear relatively straight over considerable distances, though they too can have complexities.<sup>28</sup>

Furthermore, plate boundaries often display **fractal characteristics**, meaning their irregularity and complexity are self-similar across different scales of observation.<sup>26</sup> This implies a degree of statistical roughness rather than smooth, mathematically defined curves.

Regarding **convexity**, standard Voronoi cells generated from point seeds are, by definition, convex polygons.<sup>4</sup> However, actual tectonic plates, while often appearing somewhat rounded or "blob-like" on global maps, are not perfectly convex. They are *generally* convex in the sense that they don't usually possess extreme, deep indentations relative to their overall size, but their boundaries are highly irregular and can feature significant concavities. The user's request to "form more convex structures" should be interpreted as a desire to move away from the often sharp, artificial angularity of basic Voronoi cells and to promote the emergence of more dominant, somewhat rounded macro-shapes for the plates, rather than enforcing strict mathematical convexity. This reflects a tendency for large, coherent lithospheric blocks to maintain a degree of compactness, while their edges are continuously reshaped by tectonic forces.

The procedural implication is clear: the raw output of a Voronoi tessellation is insufficient. Significant post-processing of the plate boundaries is required to introduce these naturalistic shape characteristics, including irregularity, curvature, and a more organic sense of convexity. This will be a central theme in Section 5.


## 3. Oceanic vs. Continental Plates: Fundamental Distinctions

The Earth's lithospheric plates are not homogenous; they are broadly classified into oceanic and continental types based on the nature of the crust they carry. These differences are fundamental to understanding global topography and tectonic processes.


### 3.1. Comparative Characteristics

Oceanic and continental plates differ significantly in several key physical and compositional properties:



* **Density:** This is a primary distinguishing factor. Oceanic crust is composed mainly of basalt and gabbro (mafic rocks) and is denser, with an average density of about 2.9 to 3.0 grams per cubic centimeter (g/cm³). Continental crust, being broadly granitic (felsic rocks), is less dense, with an average density of about 2.7 g/cm³.<sup>15</sup>
* **Thickness:** Oceanic crust is relatively thin, averaging only 5 to 10 kilometers (km) in thickness.<sup>15</sup> In contrast, continental crust is much thicker, averaging about 35 km, and can reach thicknesses of 70-75 km or more beneath major mountain ranges like the Himalayas.<sup>15</sup> The entire lithosphere (crust plus the rigid upper mantle) also shows this pattern: oceanic lithosphere thickness varies with age from about 6 km at mid-ocean ridges to over 100 km near subduction zones, while continental lithosphere is typically thicker, around 200 km on average.<sup>15</sup>
* **Composition:** As mentioned, oceanic crust is mafic, meaning it is richer in magnesium and iron. Continental crust is more felsic, characterized by higher concentrations of silicon and aluminum, and a greater abundance of minerals like quartz and feldspar.<sup>15</sup>
* **Age:** Oceanic crust is geologically very young. It is constantly being created at mid-ocean ridges and destroyed (recycled into the mantle) at subduction zones. Consequently, the oldest oceanic crust found in the major ocean basins is typically less than 180-200 million years old (Ma), although some remnants in enclosed seas like the eastern Mediterranean can be older, up to around 280 Ma.<sup>21</sup> Continental crust, being more buoyant and less prone to subduction, is much older and preserves a longer geological record. Some parts of the continental crust are over 4 billion years (Ga) old.<sup>21</sup>
* **Elevation and Buoyancy:** The differences in density and thickness have profound implications for isostasy (the equilibrium that exists in the Earth's crust, where thicker, less dense crustal material floats higher on the denser mantle below). Continental crust, being less dense and thicker, "floats" higher on the asthenosphere, forming the Earth's landmasses. The average surface elevation of continental crust is about 800 meters above sea level.<sup>21</sup> Conversely, the denser and thinner oceanic crust sits lower, forming the deep ocean basins, with an average depth of about 5,000 meters below sea level. Mid-ocean ridges, being sites of young, hot crust, are shallower, typically around 2,600 meters deep.<sup>21</sup> Although continental crust covers only about 40% of the Earth's surface, it accounts for approximately 70% of the total volume of the Earth's crust due to its significantly greater thickness.<sup>21</sup>

The following table summarizes these key distinctions:


<table>
  <tr>
   <td><strong>Property</strong>
   </td>
   <td><strong>Oceanic Plate Value</strong>
   </td>
   <td><strong>Continental Plate Value</strong>
   </td>
  </tr>
  <tr>
   <td>Density (g/cm³)
   </td>
   <td>~2.9 - 3.0 <sup>21</sup>
   </td>
   <td>~2.7 <sup>21</sup>
   </td>
  </tr>
  <tr>
   <td>Average Crustal Thickness (km)
   </td>
   <td>5 - 10 <sup>15</sup>
   </td>
   <td>~35 (up to 70-75 under mountains) <sup>15</sup>
   </td>
  </tr>
  <tr>
   <td>Typical Lithospheric Thickness (km)
   </td>
   <td>~6 (ridges) to >100 (old) <sup>15</sup>
   </td>
   <td>~150 - 200+ <sup>15</sup>
   </td>
  </tr>
  <tr>
   <td>General Crustal Composition
   </td>
   <td>Mafic (Basalt, Gabbro) <sup>21</sup>
   </td>
   <td>Felsic (Granite, Andesite) <sup>21</sup>
   </td>
  </tr>
  <tr>
   <td>Typical Age Range (Ma)
   </td>
   <td>0 - ~180 (oldest open ocean) <sup>21</sup>
   </td>
   <td>0 - ~4000 (4 Ga) <sup>21</sup>
   </td>
  </tr>
  <tr>
   <td>Average Elevation (rel. to sea level, m)
   </td>
   <td>~ -5000 (basin floor); ~ -2600 (ridge) <sup>21</sup>
   </td>
   <td>~ +800 <sup>21</sup>
   </td>
  </tr>
</table>


These fundamental differences are not merely descriptive; they are the primary drivers of how plates interact at their boundaries and, consequently, how they shape the Earth's largest geographical features. The simple principle that denser material tends to sink below less dense material governs much of tectonic behavior. This explains why oceanic crust readily subducts, leading to features like deep-sea trenches and volcanic arcs, while continents, being buoyant, tend to resist subduction and instead collide and build mountains. For a procedural model, accurately assigning and utilizing these plate type characteristics is therefore paramount for generating a realistic first-order global geography.


### 3.2. Impact on Plate Boundary Interactions (Focus on Subduction)

The type of crust (oceanic or continental) carried by converging lithospheric plates plays a decisive role in the nature of their interaction and the resulting geological features, particularly at subduction zones:



* **Oceanic-Continental Convergence:** When a plate carrying oceanic crust converges with a plate carrying continental crust, the denser oceanic lithosphere is thrust or pulled beneath the more buoyant continental lithosphere in a process called **subduction**.<sup>13</sup> This typically forms a deep **oceanic trench** on the seaward side of the boundary. As the subducting oceanic plate descends, it heats up and releases volatiles (like water), which causes partial melting in the overlying mantle wedge. This magma rises to the surface, leading to the formation of a chain of volcanoes, often a **volcanic mountain range** (e.g., the Andes), on the overriding continental plate, parallel to the trench.
* **Oceanic-Oceanic Convergence:** If two plates carrying oceanic crust converge, the older, colder, and therefore denser of the two plates will subduct beneath the younger, warmer, and less dense one.<sup>15</sup> Similar to oceanic-continental convergence, this forms a deep oceanic trench. The rising magma from the subducting plate erupts on the overriding oceanic plate, creating a curved chain of volcanic islands known as a **volcanic island arc** (e.g., the Mariana Islands, the Aleutian Islands).<sup>15</sup>
* **Continental-Continental Convergence:** Because continental crust is relatively thick and buoyant, neither plate subducts easily or extensively when two continental plates collide.<sup>13</sup> Instead, the immense compressional forces cause the crust to buckle, fold, fault, and thicken significantly. This process results in the formation of extensive, high-elevation **collisional mountain ranges**, such as the Himalayas, which formed (and are still forming) from the collision between the Indian and Eurasian plates.<sup>13</sup>

In the procedural generation context, these distinctions are critical. The plate_type attribute assigned to each Voronoi cell (representing a plate) will be a primary input for determining the geological outcome when these cells interact at a convergent boundary. If an "oceanic" plate meets a "continental" plate, the model should generate features consistent with subduction. If two "continental" plates meet, it should generate features consistent with a major collisional mountain belt. This rule-based approach, driven by plate type, is fundamental for creating the planet's largest and most defining geographical structures.


## 4. Initial Plate Generation on an Icosphere using Voronoi Cells

The foundation of the procedural tectonic model lies in the initial partitioning of the icosphere's surface into plate-like regions. This is achieved using Voronoi tessellation based on carefully distributed seed points.


### 4.1. Generating Seed Points (CVT Centroids)

The process commences with an icosphere, which provides a quasi-uniform spherical grid. A predetermined number of seed points, N, are initially distributed randomly across the surface of this icosphere. These points will serve as the generators for the Voronoi cells.

To ensure a more regular and evenly spaced distribution of these initial cells, which is generally preferred as a starting condition for plate generation, an iterative refinement process known as **Lloyd's algorithm** is applied.<sup>3</sup> Lloyd's algorithm works by repeatedly:



1. Constructing the Voronoi diagram based on the current set of seed points.
2. Calculating the geometric centroid (center of mass, assuming uniform density) of each Voronoi cell.
3. Moving each seed point to the calculated centroid of its own cell.

This iterative process typically converges to a **Centroidal Voronoi Tessellation (CVT)**. In a CVT, each seed point is, by definition, also the centroid of its Voronoi cell.<sup>7</sup> CVTs result in cells that are more uniform in size and shape compared to Voronoi diagrams generated from purely random points. For spherical surfaces, a common approach to approximate CVT involves iteratively computing the convex hull of the points, then the Delaunay triangulation, then the Voronoi diagram, and finally moving the seed points to the centroids of their respective Voronoi polygons.<sup>38</sup>

The number of initial seed points, N, should be chosen thoughtfully. Earth has approximately 7-15 major tectonic plates and numerous smaller ones.<sup>17</sup> Starting with N in a range like 20-60 provides flexibility, as some of these initial cells might be merged to form larger plates or fragmented to create smaller ones during subsequent processing stages, aiming to achieve a realistic plate size distribution (see Section 2.4.1).


### 4.2. Voronoi Tessellation on the Sphere

Once the positions of the N CVT centroids are finalized, the spherical Voronoi diagram is constructed. These CVT centroids act as the generator points for the tessellation. Each resulting Voronoi cell, a spherical polygon, represents an initial, raw tectonic plate on the icosphere's surface.

The generation of Voronoi diagrams on a sphere is a standard problem in computational geometry. Various libraries and algorithms exist to perform this task. For instance, Qhull is a widely used library capable of computing Delaunay triangulations and Voronoi diagrams in multiple dimensions, which can be adapted for spherical cases.<sup>1</sup> Alternatively, the method described by Burkardt <sup>39</sup>, involving the sequence of convex hull, Delaunay triangulation, and then Voronoi diagram construction, is a practical approach for spherical surfaces. The edges of these spherical Voronoi polygons are segments of great circles (or geodesics on the sphere) that are, by definition, equidistant from the two nearest seed points.<sup>2</sup>


### 4.3. Assigning Plate Types (Oceanic vs. Continental)

After the initial Voronoi plates are defined, each must be assigned a dominant type: either "oceanic" or "continental." This assignment is crucial because, as discussed in Section 3, plate type dictates fundamental properties like density, buoyancy, average elevation, and behavior at convergent boundaries. Given that most real tectonic plates are composites, containing both oceanic and continental crustal sections <sup>15</sup>, this assignment in a procedural model is an abstraction representing the plate's dominant characteristic or the nature of its core. The user's query specifically asks for a distinction between "plates that are mostly ocean and plates that are mostly land."

Several heuristic methods can be employed for this assignment:



* **Method 1: Random Assignment with Bias:**
    * A target percentage for the total global area to be covered by continental-type plates is set. Geologically, continental crust covers about 40% of Earth's surface area, including continental shelves.<sup>21</sup>
    * Plates are iterated through, and each is randomly assigned as "continental" or "oceanic" until the target continental percentage is approximated.
    * *Bias options can refine this:* Larger plates could have a higher probability of being designated continental, reflecting the tendency for continents to form large, stable cratons. Alternatively, if certain initial seed points were pre-designated as "continental cores," plates containing these seeds would be continental.
* **Method 2: "Continent" Seeding and Growth:**
    * This approach, inspired by techniques discussed in procedural generation contexts <sup>40</sup>, aims to create more contiguous and naturally shaped continental landmasses.
    * A small number of initial Voronoi cells are randomly selected to act as "continental seeds."
    * A region-growing algorithm (e.g., a breadth-first search (BFS) or a similar flood-fill type algorithm) is initiated from these seed cells.<sup>41</sup>
    * In each step of the growth process, unassigned neighboring plates adjacent to an existing continental plate are converted to "continental" type.
    * The growth process continues iteratively until the desired global percentage of continental plate area is reached.
    * All plates that remain unassigned after the growth process are designated as "oceanic."
    * This method is generally favored as it tends to produce more realistic, clustered continental masses rather than isolated "continental" cells scattered randomly across the globe.
* **Method 3: Post-Initial Elevation Assignment (More Advanced):**
    * This method would involve performing a very basic, large-scale elevation assignment across the globe first (e.g., using low-frequency Perlin noise).
    * Plates could then be classified based on their average elevation or the percentage of their area that lies above a nominal sea level. Those with significant emergent area would be typed as continental.
    * This approach is more complex to implement as an *initial* typing step because it presupposes some form of elevation data that itself depends on plate type.

The choice of typing method will significantly influence the final appearance of the generated world, particularly the distribution of land and sea. Method 2 (Continent Seeding and Growth) is often preferred for its ability to generate more organic and clustered continental forms. The plate_type attribute assigned during this stage becomes a critical input for subsequent algorithms, including elevation modeling (Section 6) and the determination of subduction dynamics at plate boundaries.

The inherent complexity of real plates, often being mosaics of both oceanic and continental sections <sup>15</sup>, is simplified in this initial stage to a dominant type. This abstraction is a practical necessity for driving distinct macroscopic behaviors in the procedural model, such as the fundamental elevation differences between land and ocean, and the critical process of subduction. While a plate might be assigned "continental," it can still possess oceanic regions, especially along its margins, and vice-versa. However, for the primary rule-based interactions, this dominant typing is key. Finer-grained distinctions of crustal type within a single plate could be introduced in more advanced stages of generation if desired.


## 5. Refining Voronoi Plate Boundaries for Geological Realism

The initial plate boundaries generated by spherical Voronoi tessellation are inherently geometric, consisting of segments of great circles that meet at sharp vertices. Real tectonic plate boundaries, however, are vastly more complex, exhibiting irregularity, fractal characteristics, and specific geologically-driven shapes. Therefore, a crucial step in procedural plate generation is the post-processing of these initial Voronoi boundaries to impart a more natural and geologically plausible appearance.


### 5.1. The Challenge: Geometric Nature of Voronoi Cells

Standard Voronoi cells, whether on a plane or a sphere, are convex polygons whose boundaries are formed by straight line segments (or their spherical equivalents, great circle arcs).<sup>4</sup> These boundaries meet at distinct vertices, often where three or more cells converge. This regular, angular geometry starkly contrasts with the observed characteristics of Earth's plate boundaries, which are shaped by immense forces, material properties, and billions of years of tectonic activity. Real boundaries can be highly irregular, jagged, smoothly arcuate (like subduction trenches <sup>15</sup>), segmented (like mid-ocean ridges), or exhibit fractal patterns.<sup>26</sup> Consequently, the direct use of unmodified Voronoi edges will result in an artificial-looking world.


### 5.2. Iterative Relaxation (e.g., Lloyd's Algorithm)

As a preliminary step before more targeted shape modifications, applying a few iterations of Lloyd's algorithm (or a variant, such as averaging the polygon corner positions to move the generating point, as suggested in some procedural generation contexts <sup>3</sup>) can be beneficial. While Lloyd's algorithm primarily aims to achieve a Centroidal Voronoi Tessellation (CVT) by making seed points coincide with cell centroids <sup>5</sup>, its application can also lead to more uniformly sized and somewhat more regularly shaped polygons.<sup>3</sup> This regularization of the initial tessellation can provide a more consistent and predictable starting point for subsequent boundary refinement algorithms, reducing extreme variations in cell size or elongation that might otherwise complicate later steps. This is a pre-conditioning step rather than one that directly imparts geological realism to the boundaries themselves.


### 5.3. Post-Processing for Enhanced Convexity and Dominance

The user's objective to create "more convex structures" is interpreted here not as achieving strict mathematical convexity (as Voronoi cells are already convex), but rather as fostering the development of larger, more dominant plates that have less angular and more organically rounded macro-shapes. Real plates, while irregular, tend towards a general compactness. This can be achieved through heuristics that simulate a form of "competition" or influence between adjacent plates.


#### 5.3.1. Boundary Node Reassignment (Plate Competition Heuristics)

This approach involves iteratively adjusting the ownership of boundary elements (vertices or small regions) based on the relative "strength" or influence of the adjacent plates.



* **User's "Plate Strength" Heuristic:** This method, proposed in the user query, provides a direct mechanism for such competition.
    1. Identify all vertices (nodes) that lie on the boundary between two or more plates.
    2. For each such boundary node, and for each plate (Pi​) adjacent to it: Calculate a "strength" value: StrengthPi​​=AreaPi​​​/Distance(node,CentroidPi​​). Where AreaPi​​ is the total area of plate Pi​, and Distance(node,CentroidPi​​) is the distance from the boundary node to the CVT centroid of plate Pi​.
    3. The boundary node (and potentially the small icosphere mesh elements, like triangles, immediately surrounding it) is then reassigned to the plate that exhibits the highest strength value at that node.
    4. This process is repeated for a set number of iterations or until the plate boundaries stabilize. This heuristic allows plates that are larger (higher Area​) or whose main mass (centroid) is closer to a contested boundary region to expand at the expense of smaller or more distant-centroid plates. This tends to smooth out very small or sharply intrusive plate features, leading to more dominant, generally convex or rounded plate shapes by effectively "absorbing" weaker neighbors or boundary segments. This concept finds analogies in ecological territory competition models <sup>45</sup> or network node importance.<sup>46</sup>
* **Alternative/Complementary Heuristics for Boundary Element Swapping:**
    * **Convexity Maximization:** Iteratively test small displacements of shared boundary vertices. If a displacement increases a chosen convexity metric (e.g., ratio of polygon area to its convex hull area, or reduction in the number/severity of concavities) for one or both adjacent plates without significantly degrading the other, the move is accepted. This draws inspiration from polygon simplification and regularization algorithms.<sup>47</sup>
    * **"Dominance" by Plate Type:** A weighting factor could be introduced into the strength calculation based on the plate type. For instance, regions of continental crust might be assigned a higher intrinsic "resistance to erosion" or boundary displacement compared to oceanic crust, making them more likely to maintain or expand their boundaries against oceanic plates in ambiguous situations.


#### 5.3.2. Filling Minor Concavities

Following the competitive reshaping process, plates might still exhibit small, sharp, or narrow concavities that appear geologically implausible for large tectonic structures. Heuristics can be applied to identify and "fill" these minor features:



1. **Identify Concavities:** Concave sections of a boundary can be identified by analyzing the sequence of internal angles between boundary segments or by comparing the polygon with its convex hull. Vertices that lie on the polygon boundary but not on its convex hull are part of a concavity.
2. **Filter by Size/Shape:** Small or shallow concavities are targeted. A concavity might be considered "minor" if its total area is below a threshold, its depth is small relative to its width (the "mouth" of the concavity), or if it's formed by only a few vertices.
3. **Fill Concavity:** If a concavity is deemed minor, it can be removed by effectively replacing the sequence of boundary segments forming the concavity with a single new segment that connects the "entry" and "exit" points of the concavity (i.e., bridging its mouth). This is conceptually similar to some polygon simplification techniques where insignificant details are removed.<sup>55</sup>

These competitive and concavity-filling steps help to transform the initial, somewhat uniform Voronoi cells into more varied and geologically plausible macro-shapes, characterized by larger, dominant plates with generally smoother and more convex-like outlines.

The process of achieving realistic plate boundaries from an initial Voronoi tessellation is inherently iterative and multi-stage. No single algorithm suffices. Instead, a pipeline of carefully chosen and ordered operations is necessary. Each algorithm in the pipeline addresses a specific aspect of the desired final plate shape: initial regularity (e.g., via Lloyd relaxation), overall dominance and large-scale convexity (via competitive heuristics like "plate strength"), fine-scale natural irregularity (via roughening techniques), and finally, selective smoothing to remove unwanted artifacts. The order of these operations is critical. For instance, applying roughening before competitive reshaping might lead to chaotic and unpredictable results, while smoothing too early might erase the beneficial effects of competition or necessary roughness. Therefore, a logical sequence, such as initial relaxation, followed by competitive reshaping, then boundary roughening, and concluding with selective smoothing, is recommended. The parameters for each stage (e.g., number of iterations for strength calculations, amplitude of noise for roughening, intensity of smoothing) will require empirical tuning to achieve the desired balance of realism and aesthetic appeal.


### 5.4. Boundary Smoothing and Roughening for Naturalistic Detail

Once the macro-shapes of the plates have been established, further refinement is needed to give their boundaries a more naturalistic texture, incorporating both smooth curves and fine-scale irregularity.


#### 5.4.1. Smoothing Techniques

These techniques are used to reduce unwanted jaggedness, sharp angles, or artifacts from previous processing steps, leading to more organic, flowing boundary lines where appropriate.



* **Chaikin's Algorithm:** This is an iterative corner-cutting algorithm highly effective for generating smooth curves.<sup>58</sup> For each segment P1-P2 on the polygon boundary, two new points Q0 and Q1 are generated. Q0 is typically placed 1/4 of the way from P1 to P2, and Q1 is placed 3/4 of the way from P1 to P2 (or equivalently, 1/4 of the way from P2 to P1). The original vertices P1, P2, etc., are discarded, and the new polygon is formed by connecting the new Q points in sequence. Applying this process iteratively results in increasingly smooth and rounded curves. It's useful for creating naturally flowing boundaries without the sharp angles of raw polygons.
* **Iterative Vertex Averaging (Laplacian Smoothing):** In this method, each vertex on a plate boundary is moved to the average position of its two neighboring vertices along that same boundary.<sup>62</sup> This process is repeated for a specified number of iterations. Laplacian smoothing effectively dampens high-frequency details (sharp jags) and tends to distribute vertices more evenly. However, it can also lead to shrinkage of the feature if not applied carefully or constrained.

These smoothing techniques should be applied judiciously, often after major shape adjustments or roughening steps, to refine the boundaries without erasing all character or essential geological forms.


#### 5.4.2. Roughening Techniques

To counteract the overly smooth or geometric lines that can result from Voronoi generation or even some smoothing processes, roughening techniques are applied to introduce natural-looking irregularity and fractal detail, which are characteristic of real plate boundaries.<sup>26</sup>



* **Constrained Noisy Edges (AmitP's Method):** This is a particularly effective technique for adding detail to polygon edges in a controlled manner, preventing overlaps between adjacent noisy boundaries.<sup>3</sup> The process for each original straight Voronoi edge is as follows:
    1. Define a quadrilateral region around the edge. The four corners of this quadrilateral are the two Voronoi cell centroids of the plates sharing the edge, and the two endpoints of the Voronoi edge itself.
    2. The original straight edge is then recursively subdivided. At each subdivision step, the midpoint of a segment is displaced randomly, but this displacement is constrained to keep the new point within the allocated quadrilateral (or sub-quadrilaterals derived from it).
    3. This creates a "noisy" or fractal-like line that replaces the original straight edge, adding detail while ensuring that the perturbed boundaries of adjacent plates do not cross each other.
* **Fractal Line Subdivision:** A simpler approach involves applying a standard fractal line generation algorithm (such as midpoint displacement with decreasing noise amplitude over iterations) to each segment of the plate boundary. The displacement magnitude should be controlled to prevent excessive distortion.

These roughening steps are crucial for breaking the artificial smoothness of purely geometric lines and imparting a more organic, complex appearance to the plate boundaries. The constrained noisy edge method is particularly valuable due to its inherent overlap prevention.


#### 5.4.3. Straightening Near-Linear Segments and Creating Arcs

After roughening, some boundary segments might be overly noisy or contain excessive detail where a simpler, more geologically plausible sweeping curve or a relatively straight segment would be more appropriate (e.g., representing long, stable transform faults or the broad, smooth arcs of some subduction zones).



* **Detecting and Simplifying Near-Linear Segments:** Algorithms like the **Douglas-Peucker algorithm** <sup>66</sup> can be used to identify and simplify sequences of boundary vertices that are nearly collinear. This algorithm recursively removes vertices that fall within a specified tolerance distance from a line segment connecting their neighbors, effectively straightening out minor jigs and jags while preserving the overall trend of the line. Alternatively, near-linearity can be detected by examining the angles between successive short segments; if a series of segments maintain a nearly consistent direction, they can be merged into a longer, single straight segment.<sup>68</sup>
* **Fitting Circular Arcs to Curved Segments:** For boundary sections that exhibit a clear, gentle curvature (often seen in subduction zones), circular arcs can be fitted to replace a sequence of many small, straight segments. This can be achieved by:
    * Selecting three well-spaced points along the curved segment and calculating the unique circle that passes through them (the circumcircle).
    * For longer, more complex curves, least-squares fitting methods can be employed to find the circular arc that best approximates a larger set of points on the boundary.<sup>70</sup> The Simplify by Straight Lines and Circular Arcs tool described in <sup>70</sup> offers parameters like Maximum Arc Angle Step and Minimum Arc Angle to control arc construction.

This final stage of refinement allows for the introduction of both highly irregular, fractal-like boundary sections and smoother, more geometrically defined (straight or arcuate) sections, reflecting the diverse morphology of real tectonic plate boundaries.

The overall goal of boundary refinement is to achieve a balance. While the user desires "more convex structures," this should not be interpreted as transforming Voronoi cells into perfect mathematical convex polygons. Real plates, though generally maintaining a coherent, somewhat compact form, are characterized by significant boundary irregularities, concavities resulting from complex interactions, and fractal details. The "plate strength" heuristic helps establish dominant, larger plates which might appear more "rounded" or generally convex by absorbing smaller, angular neighbors. However, this alone may not produce natural irregularity. Therefore, a sequence involving competitive reshaping (for dominance and general form), followed by constrained roughening (for detail and fractal character), and then selective smoothing or simplification (to remove artifacts and introduce plausible arcuate or linear sections) is necessary. The aim is to make the plates less like rigid geometric constructs and more like the "blobby" or "ameboid" shapes seen on geological maps, which reflect a long history of dynamic interaction.


## 6. Modeling Average Plate Elevation

The average elevation of a tectonic plate is a primary factor in determining whether it forms land or ocean floor. This elevation is principally governed by the type of crust (oceanic or continental) and, for oceanic plates, its age.


### 6.1. Oceanic Plate Elevation Profile

The depth of the seafloor, and thus the elevation of an oceanic plate, is strongly correlated with the age of its lithosphere. This relationship is a cornerstone of plate tectonic theory.



* **Age-Depth Relationship:** New oceanic crust is formed at mid-ocean ridges. Here, the underlying mantle is hot and buoyant, causing the ridge axis to be elevated, typically to a depth of around 2,600 meters below sea level.<sup>35</sup> As this newly formed lithosphere moves away from the spreading center, it cools conductively, contracts, and becomes denser. This increase in density causes the lithosphere to subside isostatically, leading to increasing water depth with increasing distance (and thus age) from the ridge.<sup>15</sup>
* **Mathematical Model:** The depth (D) of the seafloor at a location on a spreading mid-ocean ridge system can be approximated as a function of the age (t) of the seafloor at that point. A commonly used empirical relationship is: D(t)=Dridge​+k⋅t​ where Dridge​ is the depth at the ridge crest (e.g., ~2600 m), t is the age in millions of years, and k is an empirical constant (approximately 350 m/Ma​ for normal oceanic lithosphere).<sup>35</sup> This square-root dependence arises from models of conductive cooling of a thermal boundary layer or a cooling half-space.
* **Procedural Approach for Implementation:**
    1. **Identify Spreading Centers:** For each oceanic plate, one or more of its boundaries must be designated as "spreading centers" or mid-ocean ridges. In a procedural system where plates are generated and then assigned motion, divergent boundaries (where plates move apart) would naturally correspond to these spreading centers.
    2. **Approximate Crustal Age:** The "age" of any point (e.g., a Voronoi cell node or a point on a finer grid within the cell) on an oceanic plate can be approximated by its shortest geodesic distance to the nearest active spreading center associated with that plate. This distance can then be scaled by an average spreading rate (e.g., slow spreading ~1-2 cm/yr per side, fast spreading ~5-8 cm/yr per side <sup>34</sup>) to get a proxy for age. For example, age_proxy = distance_to_ridge / average_spreading_rate.
    3. **Calculate Elevation (Depth):** Using the age proxy, the elevation (which will be negative, representing depth) can be calculated using the age-depth relationship described above. *Snippet Relevance:* <sup>21</sup> provides average oceanic crust elevation. <sup>15</sup>, and especially <sup>35</sup> detail the age-depth relationship and ridge characteristics. <sup>33</sup> and <sup>34</sup> reiterate these points with age maps and spreading rates.

This physics-informed approach provides a strong basis for generating realistic oceanic plate topography, where vast abyssal plains deepen away from shallower mid-ocean ridge systems.


### 6.2. Continental Plate Elevation

Continental plates, due to their thicker and less dense crust, float isostatically higher on the asthenosphere compared to oceanic plates.



* **General Characteristics:** The average surface elevation of continental landmasses is approximately 800 meters above sea level.<sup>21</sup> However, continental elevation is far more varied than oceanic elevation due to a longer and more complex geological history.
* **Major Intra-plate Features influencing Average Elevation:**
    * **Shields:** These are the ancient, stable, and typically Precambrian crystalline cores of continents. They are generally characterized by low relief and relatively low average elevations, often forming extensive, flat or gently undulating plains.
    * **Platforms:** These are regions where ancient shield rock (basement) is overlain by younger, relatively flat-lying sedimentary rock layers. Platforms also tend to be relatively flat but can exhibit a wider range of elevations than exposed shields, sometimes forming broad basins or swells.
    * **Mountain Belts (Influence on Average):** While major mountain belts (e.g., Himalayas, Andes) are primarily formed at plate boundaries and represent significant deviations from average elevation, their presence can influence the overall average elevation of a continental plate if they cover a substantial portion of its area. (This report focuses on *average* plate elevation; the detailed generation of mountains is a boundary-specific phenomenon).
* **Procedural Approach for Implementation:**
    1. **Base Elevation:** Assign a base average elevation to regions designated as continental. This could be a random value within a plausible range (e.g., +200m to +1000m).
    2. **Broad-Scale Variation:** Introduce large-scale, low-frequency noise (e.g., using Perlin or Simplex noise functions) to modulate this base elevation. This can create broad regions of slightly lower, flatter terrain (simulating shields or basins) and slightly higher, more undulating terrain (simulating platforms or broad uplifts). The amplitude of this noise should be controlled to represent average variations rather than extreme topography.
    3. More significant topographic features like major mountain ranges or deep rift valleys are primarily the result of specific plate boundary interactions and would be superimposed on this average elevation profile in a later stage of terrain generation. *Snippet Relevance:* <sup>21</sup> provides the average continental surface elevation. <sup>75</sup> and <sup>77</sup> discuss continental lithospheric buoyancy and elevation models; while these models are complex for direct procedural implementation, they support the general principle of higher average elevation for continents and the influence of lithospheric properties.

This approach provides a simpler method for continental elevation compared to oceanic, reflecting the more complex and varied history of continental crust.


### 6.3. Intra-plate Elevation Variation (Basic Heuristics for Voronoi Cells)

Beyond the broad distinctions set by plate type and age, some basic intra-plate elevation variation can be introduced at the scale of the Voronoi cells themselves to prevent them from appearing perfectly flat or uniformly sloped.



* **Distance from Centroid / Boundaries:**
    * A subtle effect can be achieved by slightly modifying elevation based on the distance from a point within the cell to the cell's centroid or its boundaries. For example, elevation could gently decrease towards the boundaries of a cell, making the plate center a slight high point, or vice-versa. This can give a subtle "domed" or "basin" character to individual plates.
* **Noise Functions:**
    * Apply a low-amplitude, medium-frequency noise function (e.g., Perlin, Simplex noise) across the surface of each plate. The characteristics of this noise can be tuned based on the plate type:
        * For **continental plates**, this noise can simulate rolling hills, broad valleys, or the general undulation of shield and platform areas.
        * For **oceanic plates**, this noise can represent abyssal hills, seamount provinces (if not modeled separately by hotspots), or other irregularities on the seafloor, superimposed on the larger age-depth trend. The amplitude of this noise should generally be smaller than the first-order elevation differences between oceanic and continental plates or major boundary features. *Snippet Relevance:* General procedural terrain generation techniques often use noise functions for such details.<sup>7378</sup> explicitly suggests a cellModifier function that could be used to alter a base heightmap within each Voronoi cell, for example, by blending with a noise function. <sup>79</sup> discusses blending based on proximity to cell centers/boundaries.

These intra-plate variations add a layer of detail and prevent the generated plates from appearing unnaturally uniform before more significant features from boundary interactions are added.

The generation of realistic planetary elevation is inherently a hierarchical process, reflecting geological mechanisms operating at different scales.



1. **First-Order Elevation:** Primarily determined by the fundamental distinction between oceanic and continental crust (buoyancy, thickness <sup>21</sup>) and, for oceanic crust, its age (thermal subsidence <sup>35</sup>). This establishes the basic land-sea distribution and the general depth of ocean basins versus the elevation of continents.
2. **Second-Order Features:** These are the major topographic expressions created by plate boundary interactions, such as mountain ranges at convergent boundaries, deep oceanic trenches at subduction zones, and rift valleys at divergent boundaries. (While not the primary focus of *this* report section, they are a critical component of the overall world generation that the user will address).
3. **Third-Order Features:** These include smaller-scale intra-plate variations such as regional uplifts or basins on continents <sup>75</sup>, abyssal hills on the ocean floor, volcanic seamounts (some related to hotspots), and general terrain roughness.

This section has focused on modeling the first-order average plate elevations and introducing simple third-order variations using noise. The user's subsequent implementation of plate boundary interactions will be responsible for creating the dramatic second-order features. This layered approach ensures that the foundational elevations are geologically sound before more detailed topography is sculpted.


## 7. Pseudo-code Algorithms for Implementation

This section provides simplified pseudo-code algorithms for the core processes discussed. These are intended as a guide for an AI code generator or for manual implementation, focusing on the logic rather than specific programming language syntax. Each Voronoi cell is considered a "plate" and has properties like ID, type (CONTINENTAL/OCEANIC), centroid, boundary_nodes, euler_pole_latitude, euler_pole_longitude, angular_velocity, area, etc. Nodes are vertices on the icosphere mesh that define plate boundaries.


### 7.1. Algorithm 1: Initial Plate Generation

Code snippet

FUNCTION GenerateInitialPlates(icosphere_mesh, num_plate_seeds, num_lloyd_iterations, continental_percentage_target, continent_seed_method) \
    // Step 1: Generate initial seed points on the sphere \
    initial_seeds = GenerateRandomPointsOnSphere(icosphere_mesh, num_plate_seeds) \
 \
    // Step 2: Apply Lloyd's Relaxation to get CVT centroids \
    // This involves iteratively recomputing Voronoi cells and moving seeds to centroids \
    // See [5, 9, 38, 39] for details on Lloyd's algorithm \
    cvt_centroids = initial_seeds \
    FOR i = 1 TO num_lloyd_iterations \
        temp_voronoi_cells = ComputeSphericalVoronoi(cvt_centroids, icosphere_mesh) \
        new_centroids = \
        FOR EACH cell IN temp_voronoi_cells \
            ADD CalculateCellCentroid(cell, icosphere_mesh) TO new_centroids \
        END FOR \
        cvt_centroids = new_centroids \
    END FOR \
 \
    // Step 3: Compute final Voronoi tessellation from CVT centroids \
    // Each cell in voronoi_cells is a list of vertex coordinates defining a spherical polygon \
    voronoi_cells_polygons = ComputeSphericalVoronoi(cvt_centroids, icosphere_mesh) // [2] \
 \
    // Step 4: Create plate objects from Voronoi cells \
    plates = \
    FOR EACH polygon_vertices IN voronoi_cells_polygons \
        plate = new Plate() \
        plate.ID = GenerateUniqueID() \
        plate.centroid = GetCentroidFromPolygon(polygon_vertices) // This should be close to the CVT seed \
        plate.boundary_nodes = polygon_vertices // Nodes are vertices of the Voronoi cell \
        plate.area = CalculateSphericalPolygonArea(polygon_vertices, icosphere_mesh.radius) \
        ADD plate TO plates \
    END FOR \
 \
    // Step 5: Assign plate types (Oceanic vs. Continental) \
    // Method: "SEED_AND_GROW" is recommended for more natural continent distribution [40, 41] \
    // Earth's continental crust covers ~40% of surface [21] \
    IF continent_seed_method == "SEED_AND_GROW" \
        num_continent_seeds = MAX(1, ROUND(num_plate_seeds * 0.1)) // e.g., 10% of plates as initial seeds \
        seed_indices = RandomlySelectNIndices(length(plates), num_continent_seeds) \
 \
        FOR EACH plate IN plates \
            plate.type = OCEANIC // Default to oceanic \
        END FOR \
 \
        queue = \
        total_continental_area = 0 \
        target_total_area = CalculateTotalSphereArea(icosphere_mesh) * continental_percentage_target \
 \
        FOR EACH index IN seed_indices \
            plates[index].type = CONTINENTAL \
            total_continental_area += plates[index].area \
            ADD plates[index] TO queue \
        END FOR \
 \
        WHILE length(queue) > 0 AND total_continental_area &lt; target_total_area \
            current_plate = REMOVE_FIRST(queue) \
            FOR EACH neighbor_plate IN FindAdjacentPlates(current_plate, plates) // Needs adjacency info \
                IF neighbor_plate.type == OCEANIC \
                    neighbor_plate.type = CONTINENTAL \
                    total_continental_area += neighbor_plate.area \
                    ADD neighbor_plate TO queue \
                    IF total_continental_area >= target_total_area THEN BREAK // Exit inner loop \
                END IF \
            END FOR \
        END WHILE \
    ELSE // Fallback to simple random assignment with bias \
        num_continental_target = ROUND(length(plates) * continental_percentage_target) \
        num_continental_assigned = 0 \
        // Potentially sort plates by area and assign larger ones as continental first \
        FOR EACH plate IN plates \
            IF num_continental_assigned &lt; num_continental_target // Simplified random assignment \
                IF RandomChance(continental_percentage_target) // or more sophisticated bias \
                    plate.type = CONTINENTAL \
                    num_continental_assigned += 1 \
                ELSE \
                    plate.type = OCEANIC \
                END IF \
            ELSE \
                plate.type = OCEANIC \
            END IF \
        END FOR \
    END IF \
 \
    RETURN plates \
END FUNCTION \



### 7.2. Algorithm 2: Plate Boundary Refinement

Code snippet

FUNCTION RefinePlateBoundaries(plates, strength_iterations, chaikin_smoothing_iterations, noisy_edge_subdivisions, noisy_edge_displacement_factor) \
    // This function assumes plates have defined boundaries (sequences of nodes) \
    // and adjacency information is available or can be computed. \
 \
    // Stage 1: Plate Competition (User's "Plate Strength" Heuristic) \
    // This stage aims to make some plates more dominant and boundaries less angular. \
    // It involves reassigning ownership of boundary nodes (or small mesh elements near them). \
    // This is a complex step involving geometric tests and data structure updates for boundary nodes. \
    // The core idea is: \
    FOR iter_strength = 1 TO strength_iterations \
        boundary_node_transfers = // Store (node_to_move, new_owner_plate) \
 \
        FOR EACH plate_A IN plates \
            FOR EACH boundary_node IN plate_A.boundary_nodes \
                // A boundary node is shared by at least two plates. \
                // Identify all plates sharing this node. \
                adjacent_plates_to_node = GetPlatesSharingNode(boundary_node, plates) \
                IF length(adjacent_plates_to_node) &lt; 2 THEN CONTINUE // Node not on a boundary \
 \
                winning_plate = NULL \
                max_strength = -1 \
 \
                FOR EACH plate_B IN adjacent_plates_to_node \
                    // Calculate strength based on user's formula (sqrt(Area) / distance_to_centroid) \
                    dist_to_centroid = SphericalDistance(boundary_node.position, plate_B.centroid) \
                    IF dist_to_centroid == 0 THEN dist_to_centroid = 0.001 // Avoid division by zero \
                    strength = SQRT(plate_B.area) / dist_to_centroid \
 \
                    IF strength > max_strength \
                        max_strength = strength \
                        winning_plate = plate_B \
                    END IF \
                END FOR \
 \
                // If the node's current owner is not the winning_plate, mark for transfer. \
                // This requires careful management of which plate "owns" which segment of the boundary. \
                // A simpler approach might be to consider small mesh triangles around the node. \
                // For this pseudocode, we'll conceptualize it as node transfer. \
                current_owner_plate = GetCurrentOwnerOfNodeSegment(boundary_node, plate_A) // Simplified \
                IF winning_plate!= current_owner_plate AND winning_plate!= NULL \
                    ADD (boundary_node, winning_plate) TO boundary_node_transfers \
                END IF \
            END FOR \
        END FOR \
 \
        // Apply transfers (this is the hard part: updating shared boundaries) \
        FOR EACH transfer IN boundary_node_transfers \
            node_to_move = transfer.node \
            new_owner_plate = transfer.plate_object \
            old_owner_plate = GetOldOwnerPlate(node_to_move, plates, new_owner_plate) // Find the other plate(s) \
            // Update boundary data structures of new_owner_plate and old_owner_plate \
            // This involves removing the node from old_owner's boundary segment and adding to new_owner's, \
            // potentially creating new edges or merging existing ones. \
            // This is highly dependent on the specific mesh and boundary representation. \
            // For simplicity, assume a function handles this complex update: \
            UpdateBoundaryNodeOwnership(node_to_move, new_owner_plate, old_owner_plate, plates) \
        END FOR \
        RecalculatePlateAreasAndBoundaries(plates) // After all transfers in an iteration \
    END FOR \
 \
    // Stage 2: Boundary Roughening (Constrained Noisy Edges [3]) \
    new_plate_boundaries = {} // Dictionary: plate_id -> list of new boundary segments \
    FOR EACH plate IN plates \
        new_plate_boundaries = \
        FOR EACH original_segment IN GetBoundarySegments(plate) // A segment is (node1, node2) \
            node1 = original_segment.start_node \
            node2 = original_segment.end_node \
            plate_center1 = plate.centroid \
            neighbor_plate = GetNeighborAcrossSegment(plate, original_segment, plates) \
            IF neighbor_plate == NULL THEN // Edge of the world (not applicable for closed sphere) \
                ADD original_segment.vertices TO new_plate_boundaries // Keep as is \
                CONTINUE \
            END IF \
            plate_center2 = neighbor_plate.centroid \
 \
            // Define the quadrilateral for constraining noise [3] \
            // Points: node1, node2, plate_center1, plate_center2 \
            // The noisy line will replace the direct edge node1-node2 \
            noisy_vertices = GenerateNoisyEdgeRecursive(node1.position, node2.position, \
                                                     plate_center1, plate_center2, \
                                                     noisy_edge_subdivisions, noisy_edge_displacement_factor) \
            ADD noisy_vertices TO new_plate_boundaries \
        END FOR \
    END FOR \
    // Update all plate boundaries simultaneously with the new_plate_boundaries \
    UpdateAllPlateBoundariesFromSegments(plates, new_plate_boundaries) \
 \
 \
    // Stage 3: Selective Smoothing (Chaikin's Algorithm [58, 59]) \
    FOR iter_smooth = 1 TO chaikin_smoothing_iterations \
        FOR EACH plate IN plates \
            current_boundary_vertices = plate.GetOrderedBoundaryVertices() \
            IF length(current_boundary_vertices) &lt; 3 THEN CONTINUE \
 \
            new_boundary_vertices = \
            num_verts = length(current_boundary_vertices) \
            FOR i = 0 TO num_verts - 1 \
                p0 = current_boundary_vertices[i] \
                p1 = current_boundary_vertices[(i + 1) % num_verts] // Wrap around for closed polygon \
 \
                // Chaikin: new points are 1/4 and 3/4 along the segment \
                q0_x = 0.75 * p0.x + 0.25 * p1.x \
                q0_y = 0.75 * p0.y + 0.25 * p1.y \
                q0_z = 0.75 * p0.z + 0.25 * p1.z // Assuming 3D coordinates \
                ADD new Vertex(q0_x, q0_y, q0_z) TO new_boundary_vertices \
 \
                q1_x = 0.25 * p0.x + 0.75 * p1.x \
                q1_y = 0.25 * p0.y + 0.75 * p1.y \
                q1_z = 0.25 * p0.z + 0.75 * p1.z \
                ADD new Vertex(q1_x, q1_y, q1_z) TO new_boundary_vertices \
            END FOR \
            plate.SetBoundaryVertices(new_boundary_vertices) // Update plate with smoothed boundary \
        END FOR \
    END FOR \
 \
    // Optional: Further steps like filling minor concavities [56, 57] \
    // or straightening near-linear segments / fitting arcs [66, 70] \
    // would follow here, each with their own iterative logic. \
 \
    RETURN plates \
END FUNCTION \
 \
// Helper for Noisy Edges (Recursive part) \
FUNCTION GenerateNoisyEdgeRecursive(v1, v2, c1, c2, subdivisions_left, displacement_factor) \
    IF subdivisions_left == 0 \
        RETURN [v1, v2] // Base case: return the segment itself \
    END IF \
 \
    midpoint = (v1 + v2) / 2 \
    // Displacement should be perpendicular to segment v1-v2 and constrained within the quad (v1,c1,v2,c2) \
    // This is a simplification; true constrained displacement is more complex. \
    // For a sphere, displacement should be along the surface. \
    displacement_vector = PerpendicularVectorOnSphere(v1, v2) * Random(-1, 1) * displacement_factor \
    displaced_midpoint = ProjectOntoSphere(midpoint + displacement_vector) \
 \
    // Ensure displaced_midpoint stays within the conceptual quadrilateral \
    // This requires a robust geometric check (e.g., point in spherical quadrilateral) \
    // For simplicity, this check is omitted here but is crucial.[3] \
    // displaced_midpoint = ConstrainToQuad(displaced_midpoint, v1, v2, c1, c2) \
 \
    path1 = GenerateNoisyEdgeRecursive(v1, displaced_midpoint, c1, c2, subdivisions_left - 1, displacement_factor * 0.5) \
    path2 = GenerateNoisyEdgeRecursive(displaced_midpoint, v2, c1, c2, subdivisions_left - 1, displacement_factor * 0.5) \
 \
    RETURN Concatenate(path1.RemoveLast(), path2) // Avoid duplicating midpoint \
END FUNCTION \



### 7.3. Algorithm 3: Assigning Plate Velocities and Rotations

Code snippet

FUNCTION AssignPlateMotion(plates) \
    // Assign a random Euler pole and angular velocity to each plate [24, 25] \
    FOR EACH plate IN plates \
        // Euler pole: random point on the sphere \
        plate.euler_pole_latitude = RandomFloat(-90.0, 90.0)  // Degrees \
        plate.euler_pole_longitude = RandomFloat(-180.0, 180.0) // Degrees \
 \
        // Angular velocity: realistic range, e.g., 0.1 to 2.0 deg/Myr \
        // Based on NUVEL-1A values [25] and general plate speeds [15] \
        // Max observed is ~10-16 cm/yr. 1 deg/Myr at Earth's equator ~ 11 cm/yr. \
        // So, a range like 0.1 to 1.5 deg/Myr seems reasonable. \
        plate.angular_velocity_deg_per_myr = RandomFloat(0.1, 1.5) \
 \
        // Store as radians per unit time if preferred for calculations \
        // plate.angular_velocity_rad_per_myr = DEGREES_TO_RADIANS(plate.angular_velocity_deg_per_myr) \
    END FOR \
    RETURN plates \
END FUNCTION \



### 7.4. Algorithm 4: Calculating Average Plate Elevations

Code snippet

FUNCTION CalculateAveragePlateElevations(plates, earth_radius) \
    // Assigns an average elevation to each node/point within a plate \
    // based on plate type and, for oceanic, age proxy. \
 \
    FOR EACH plate IN plates \
        IF plate.type == CONTINENTAL \
            // Base elevation for continental crust, e.g., +200m to +800m [21] \
            base_elevation_continental = RandomFloat(200.0, 800.0) \
            continental_noise_amplitude = RandomFloat(50.0, 300.0) // For intra-plate variation \
 \
            FOR EACH node IN plate.GetNodes() // Assuming nodes are vertices of the icosphere mesh belonging to this plate \
                // Add low-frequency noise for broad variations (shields, platforms) \
                noise_value = PerlinNoise2D(node.position_on_sphere.x, node.position_on_sphere.y) // Or 3D noise \
                node.elevation = base_elevation_continental + noise_value * continental_noise_amplitude \
            END FOR \
 \
        ELSE // plate.type == OCEANIC \
            // Identify spreading ridge(s) for this plate. This is a simplification. \
            // A more robust method would use actual divergent boundaries from plate motion. \
            // For this pseudo-code, assume 'plate.spreading_centers' is a list of points on ridges. \
            // If not available, one might pick a boundary segment or a point on the plate as a proxy. \
            // Spreading rate in meters per year (e.g., 0.01 to 0.1 m/yr, i.e., 1-10 cm/yr [34]) \
            plate.spreading_rate_m_per_yr = RandomFloat(0.01, 0.1) \
            oceanic_noise_amplitude = RandomFloat(50.0, 200.0) // For abyssal hills \
 \
            // If no explicit spreading centers, use a heuristic (e.g., farthest point from subduction zone, or a random edge) \
            IF IsEmpty(plate.spreading_centers) \
                // Fallback: could approximate by finding a point on the boundary that is \
                // part of a divergent boundary if boundary types are known, or a random boundary point. \
                // For simplicity, we'll skip detailed fallback here. \
                // A simple proxy: use distance from plate centroid as a very rough age proxy (younger near centroid if it's a small new plate) \
            END IF \
 \
            FOR EACH node IN plate.GetNodes() \
                IF IsEmpty(plate.spreading_centers) \
                    // Simplified fallback if no ridges defined for this plate (less realistic) \
                    node.elevation = -4500.0 // Default abyssal plain depth \
                ELSE \
                    min_distance_to_ridge = PositiveInfinity() \
                    FOR EACH ridge_point IN plate.spreading_centers \
                        dist = SphericalDistance(node.position_on_sphere, ridge_point.position_on_sphere) \
                        min_distance_to_ridge = MIN(min_distance_to_ridge, dist) \
                    END FOR \
 \
                    // Age proxy in Million Years (Myr) \
                    // distance (m) / rate (m/yr) = years; years / 1,000,000 = Myr \
                    age_proxy_myr = (min_distance_to_ridge / plate.spreading_rate_m_per_yr) / 1000000.0 \
                    age_proxy_myr = MAX(age_proxy_myr, 0.1) // Avoid sqrt(0) and ensure minimum age \
 \
                    depth_at_ridge_m = -2600.0 // [35] \
                    subsidence_constant_k = 350.0 // m / sqrt(Myr) [35] \
 \
                    node.elevation = depth_at_ridge_m - subsidence_constant_k * SQRT(age_proxy_myr) \
                END IF \
 \
                // Add smaller scale noise for abyssal hills \
                noise_value = PerlinNoise2D(node.position_on_sphere.x, node.position_on_sphere.y) \
                node.elevation += noise_value * oceanic_noise_amplitude \
                node.elevation = MAX(node.elevation, -8000.0) // Cap max depth (trenches are deeper, boundary features) \
            END FOR \
        END IF \
    END FOR \
    RETURN plates \
END FUNCTION \


The successful implementation of these algorithms relies on a modular design. Each distinct aspect of plate generation—initial tessellation, type assignment, boundary refinement, motion, and elevation—can be encapsulated in separate functions or modules. This approach facilitates development, testing, and future expansions. Many of the proposed steps, particularly in boundary refinement (like Lloyd relaxation, the "plate strength" heuristic, and Chaikin smoothing), are iterative by nature.<sup>5</sup> The number of iterations and other control parameters (e.g., strength factors, displacement magnitudes, smoothing passes) will require careful tuning and experimentation by the user to achieve the desired visual and geological realism in the final procedurally generated world.


## 8. Conclusion and Further Considerations


### Recap of the Proposed Model

This report has detailed a geologically-informed procedural model for generating tectonic plates on an icosphere using Voronoi cells. The key stages of this model are:



1. **Initial Plate Generation:** Creation of seed points on the icosphere, refined using Lloyd's algorithm to achieve a Centroidal Voronoi Tessellation (CVT). Spherical Voronoi diagrams are then computed from these CVT centroids, with each cell representing an initial plate.
2. **Plate Typing:** Assignment of a dominant type (oceanic or continental) to each plate, preferably using a continent seeding and growth heuristic to achieve realistic clustering of landmasses.
3. **Boundary Refinement:** A multi-stage process to transform the geometric Voronoi boundaries into more natural forms. This includes:
    * Iterative boundary node reassignment based on a "plate strength" heuristic to create more dominant, generally convex plate shapes.
    * Constrained noisy edge generation to introduce fractal-like irregularity.
    * Selective smoothing (e.g., Chaikin's algorithm) to reduce unwanted angularity.
    * Optional steps like filling minor concavities and simplifying near-linear segments or fitting arcs.
4. **Motion Assignment:** Definition of plate movement by assigning a random Euler pole and angular velocity to each plate, allowing for the calculation of velocity vectors across the plate surface.
5. **Average Elevation Assignment:** Calculation of average plate elevation based on type. Oceanic plates follow an age-depth relationship (depth proportional to the square root of age, proxied by distance from spreading centers). Continental plates receive a higher base elevation, with broad variations introduced by noise functions. Basic intra-plate noise adds further detail.


### Strengths

The proposed model offers several strengths for procedural world generation:



* **Balances Realism and Efficiency:** It incorporates key geological principles (density differences, age-depth subsidence, Euler rotations, boundary interactions) without resorting to computationally prohibitive full geodynamic simulations.
* **Structured and Modular Approach:** The generation process is broken down into logical, manageable stages, facilitating implementation and tuning.
* **Addresses User Requirements:** It directly addresses the user's requests for modeling plate size, shape, velocity, rotation, oceanic/continental distinctions, average elevation, and methods for refining Voronoi boundaries (including the "plate strength" concept).
* **Foundation for Further Detail:** The generated plates provide a solid and geologically plausible foundation upon which more detailed geographical features (specific mountain types, rift valleys, volcanic arcs, erosion patterns) can be subsequently layered.


### Limitations

Despite its strengths, the model has inherent simplifications and limitations:



* **Static Generation:** The model generates a snapshot of tectonic plates. It does not simulate the dynamic evolution of plates over geological time (e.g., continental drift, supercontinent cycles, changes in plate boundaries).
* **Simplified Plate Interiors:** Internal plate deformation, intra-plate volcanism (hotspots, though they could be added), and complex stress fields within plates are not explicitly modeled.
* **Abstracted Boundary Interactions:** While plate types inform subduction potential, the detailed physics and diverse morphologies of specific plate boundary types (e.g., different styles of subduction, transform fault complexities, rift valley evolution) are simplified in this foundational model.
* **Homogenized Plate Types:** Real plates are often mosaics of oceanic and continental crust. The model simplifies this by assigning a dominant type, which is a necessary abstraction for initial large-scale differentiation.
* **Parameter Tuning:** Achieving desired results will likely require significant experimentation and tuning of various parameters (number of iterations, strength factors, noise amplitudes, etc.).


### Potential Areas for Future Expansion

The framework presented here can be extended by the user to incorporate more sophisticated geological realism:



* **Dynamic Plate Evolution:** Implement a time-stepping simulation where plates move according to their Euler vectors, and boundaries interact, leading to rifting, subduction, collision, and changes in plate configuration over simulated geological time.
* **Detailed Plate Boundary Features:** Develop specific modules to generate characteristic topography and geology for different types of plate boundaries (e.g., accretionary wedges and volcanic arcs at subduction zones, axial valleys and abyssal hills at spreading ridges, strike-slip faults and pull-apart basins at transform boundaries).
* **Mantle Plumes and Hotspots:** Introduce mantle plumes that can create volcanic island chains (like Hawaii) or large igneous provinces, independent of plate boundaries, by allowing them to "burn through" overlying plates.
* **Crustal Thickness and Isostasy:** Model crustal thickness more explicitly, especially for continental plates, and use more detailed isostatic calculations to refine elevation.
* **Finer-Grained Crustal Type Distribution:** Allow individual plates to be composed of distinct regions of oceanic and continental crust, potentially generated through a history of rifting and accretion. This would enable more complex boundary interactions, such as an oceanic part of a plate subducting while its continental part collides.
* **Erosion and Sedimentation:** Once initial topography is established, apply procedural erosion and sedimentation models to further shape the landscape realistically.

By building upon the geologically-informed foundation provided by this report, the user can create increasingly detailed and dynamic procedural worlds that reflect the fascinating complexity of Earth's tectonic system.


#### Works cited



1. voronoi tessellation method: Topics by Science.gov, accessed May 17, 2025, [https://www.science.gov/topicpages/v/voronoi+tessellation+method](https://www.science.gov/topicpages/v/voronoi+tessellation+method)
2. (PDF) Voronoi tessellation on the ellipsoidal earth for vector data - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/323211137_Voronoi_tessellation_on_the_ellipsoidal_earth_for_vector_data](https://www.researchgate.net/publication/323211137_Voronoi_tessellation_on_the_ellipsoidal_earth_for_vector_data)
3. Polygonal Map Generation for Games - Stanford, accessed May 17, 2025, [http://www-cs-students.stanford.edu/~amitp/game-programming/polygon-map-generation/](http://www-cs-students.stanford.edu/~amitp/game-programming/polygon-map-generation/)
4. Voronoi diagram - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Voronoi_diagram](https://en.wikipedia.org/wiki/Voronoi_diagram)
5. The Fascinating World of Voronoi Diagrams - Built In, accessed May 17, 2025, [https://builtin.com/data-science/voronoi-diagram](https://builtin.com/data-science/voronoi-diagram)
6. Voronoi Diagrams - ScholarWorks, accessed May 17, 2025, [https://scholarworks.umt.edu/cgi/viewcontent.cgi?article=1009&context=tme](https://scholarworks.umt.edu/cgi/viewcontent.cgi?article=1009&context=tme)
7. Hyperbolic Centroidal Voronoi Tessellation - The University of Texas at Dallas, accessed May 17, 2025, [https://www.utdallas.edu/~xguo/HCVT.pdf](https://www.utdallas.edu/~xguo/HCVT.pdf)
8. Intrinsic computation of centroidal Voronoi tessellation (CVT) on ..., accessed May 17, 2025, [https://www.researchgate.net/publication/265090403_Intrinsic_computation_of_centroidal_Voronoi_tessellation_CVT_on_meshes](https://www.researchgate.net/publication/265090403_Intrinsic_computation_of_centroidal_Voronoi_tessellation_CVT_on_meshes)
9. Fast Methods for Computing Centroidal Voronoi Tessellations - UCI Mathematics, accessed May 17, 2025, [https://www.math.uci.edu/~chenlong/Papers/CVT.pdf](https://www.math.uci.edu/~chenlong/Papers/CVT.pdf)
10. Terrain Generation 3: Voronoi Diagrams - LeatherBee Games, accessed May 17, 2025, [https://leatherbee.org/index.php/2018/10/06/terrain-generation-3-voronoi-diagrams/](https://leatherbee.org/index.php/2018/10/06/terrain-generation-3-voronoi-diagrams/)
11. Plate Tectonics—What Are the Forces that Drive Plate Tectonics ..., accessed May 17, 2025, [https://www.iris.edu/hq/inclass/animation/what_are_the_forces_that_drive_plate_tectonics](https://www.iris.edu/hq/inclass/animation/what_are_the_forces_that_drive_plate_tectonics)
12. Plate Tectonics - Understanding Global Change, accessed May 17, 2025, [https://ugc.berkeley.edu/background-content/plate-tectonics/](https://ugc.berkeley.edu/background-content/plate-tectonics/)
13. Continental Movement by Plate Tectonics | manoa.hawaii.edu/ExploringOurFluidEarth, accessed May 17, 2025, [https://manoa.hawaii.edu/exploringourfluidearth/physical/ocean-floor/continental-movement-plate-tectonics](https://manoa.hawaii.edu/exploringourfluidearth/physical/ocean-floor/continental-movement-plate-tectonics)
14. Types of plate boundaries and their characteristics | Earth Systems Science Class Notes, accessed May 17, 2025, [https://library.fiveable.me/earth-systems-science/unit-3/types-plate-boundaries-characteristics/study-guide/DPlbh9jUTvV35T7q](https://library.fiveable.me/earth-systems-science/unit-3/types-plate-boundaries-characteristics/study-guide/DPlbh9jUTvV35T7q)
15. Plate tectonics - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Plate_tectonics](https://en.wikipedia.org/wiki/Plate_tectonics)
16. What features form at plate tectonic boundaries? - NOAA Ocean Exploration, accessed May 17, 2025, [https://oceanexplorer.noaa.gov/facts/tectonic-features.html](https://oceanexplorer.noaa.gov/facts/tectonic-features.html)
17. THIS DYNAMIC PLANET: A TEACHING COMPANION - USGS.gov, accessed May 17, 2025, [https://volcanoes.usgs.gov/vsc/file_mngr/file-139/This_Dynamic_Planet-Teaching_Companion_Packet.pdf](https://volcanoes.usgs.gov/vsc/file_mngr/file-139/This_Dynamic_Planet-Teaching_Companion_Packet.pdf)
18. Plate Tectonics Map - USGS.gov, accessed May 17, 2025, [https://www.usgs.gov/media/images/platetectonicsgif](https://www.usgs.gov/media/images/platetectonicsgif)
19. The Tectonic Challenge | NOAA, accessed May 17, 2025, [https://www.noaa.gov/tectonic-challenge-hands-on-plate-tectonics-activities](https://www.noaa.gov/tectonic-challenge-hands-on-plate-tectonics-activities)
20. Exploring Plate Tectonics | www.manoa.hawaii.edu/sealearning, accessed May 17, 2025, [https://manoa.hawaii.edu/sealearning/grade-4/earth-and-space-science/exploring-plate-tectonics](https://manoa.hawaii.edu/sealearning/grade-4/earth-and-space-science/exploring-plate-tectonics)
21. www.earthdate.org, accessed May 17, 2025, [https://www.earthdate.org/files/000/004/286/EarthDate_411_C.pdf](https://www.earthdate.org/files/000/004/286/EarthDate_411_C.pdf)
22. How fast do tectonic plates move? | U.S. Geological Survey - USGS.gov, accessed May 17, 2025, [https://www.usgs.gov/faqs/how-fast-do-tectonic-plates-move](https://www.usgs.gov/faqs/how-fast-do-tectonic-plates-move)
23. Simple Euler Poles | Seth Stein - Northwestern University, accessed May 17, 2025, [https://sites.northwestern.edu/sethstein/simple-euler-poles/](https://sites.northwestern.edu/sethstein/simple-euler-poles/)
24. Plate motion : Euler pole • Geological model : Nuvel-1A, accessed May 17, 2025, [https://www.geologie.ens.fr/~vigny/cours/GSoCAS-4.pdf](https://www.geologie.ens.fr/~vigny/cours/GSoCAS-4.pdf)
25. Untitled - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/profile/C-Demets/publication/227822173_Current_Plate_Motions/links/5d0234ee92851c874c6275e6/Current-Plate-Motions.pdf](https://www.researchgate.net/profile/C-Demets/publication/227822173_Current_Plate_Motions/links/5d0234ee92851c874c6275e6/Current-Plate-Motions.pdf)
26. GSA Today - Broken Sheets—On the Numbers and Areas of Tectonic Plates, accessed May 17, 2025, [https://www.geosociety.org/gsatoday/science/G358A/article.htm](https://www.geosociety.org/gsatoday/science/G358A/article.htm)
27. Effect of margin curvature on plate deformation in a 3-D numerical model of subduction zones | Request PDF - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/233807659_Effect_of_margin_curvature_on_plate_deformation_in_a_3-D_numerical_model_of_subduction_zones](https://www.researchgate.net/publication/233807659_Effect_of_margin_curvature_on_plate_deformation_in_a_3-D_numerical_model_of_subduction_zones)
28. What are the different types of plate tectonic boundaries? - NOAA Ocean Exploration, accessed May 17, 2025, [https://oceanexplorer.noaa.gov/facts/plate-boundaries.html](https://oceanexplorer.noaa.gov/facts/plate-boundaries.html)
29. FRACTAL PROPERTIES OF THE ELEMENTS OF PLATE TECTONICS - Минно-геоложки университет „Св. Иван Рилски“, accessed May 17, 2025, [https://mgu.bg/wp-content/uploads/2021/10/B1_RangelovB_IvanovY-2017.pdf](https://mgu.bg/wp-content/uploads/2021/10/B1_RangelovB_IvanovY-2017.pdf)
30. A Review of Properties and Variations of Voronoi Diagrams, accessed May 17, 2025, [https://www.whitman.edu/Documents/Academics/Mathematics/dobrinat.pdf](https://www.whitman.edu/Documents/Academics/Mathematics/dobrinat.pdf)
31. www.britannica.com, accessed May 17, 2025, [https://www.britannica.com/science/continental-crust#:~:text=Continental%20crust%20is%20broadly%20granitic,3%20grams%20per%20cubic%20cm.](https://www.britannica.com/science/continental-crust#:~:text=Continental%20crust%20is%20broadly%20granitic,3%20grams%20per%20cubic%20cm.)
32. This Dynamic Planet Map - Global Volcanism Program, accessed May 17, 2025, [https://volcano.si.edu/resource_dynamicplanet.cfm](https://volcano.si.edu/resource_dynamicplanet.cfm)
33. 19.2: The Geology of the Oceanic Crust - Geosciences LibreTexts, accessed May 17, 2025, [https://geo.libretexts.org/Courses/Sierra_College/Physical_Geology_(Sierra_College_Edition)/19%3A_Geology_of_the_Oceans/19.02%3A_The_Geology_of_the_Oceanic_Crust](https://geo.libretexts.org/Courses/Sierra_College/Physical_Geology_(Sierra_College_Edition)/19%3A_Geology_of_the_Oceans/19.02%3A_The_Geology_of_the_Oceanic_Crust)
34. 18.2 The Geology of the Oceanic Crust - BC Open Textbooks, accessed May 17, 2025, [https://opentextbc.ca/geology/chapter/18-2-the-geology-of-the-oceanic-crust/](https://opentextbc.ca/geology/chapter/18-2-the-geology-of-the-oceanic-crust/)
35. Mid-ocean ridge - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Mid-ocean_ridge](https://en.wikipedia.org/wiki/Mid-ocean_ridge)
36. Weighted Voronoi Stippling / John C - Observable, accessed May 17, 2025, [https://observablehq.com/@real-john-cheung/weighted-voronoi-stippling](https://observablehq.com/@real-john-cheung/weighted-voronoi-stippling)
37. Lloyd Relaxation of Voronoi Diagrams - Wolfram Demonstrations Project, accessed May 17, 2025, [https://demonstrations.wolfram.com/LloydRelaxationOfVoronoiDiagrams/](https://demonstrations.wolfram.com/LloydRelaxationOfVoronoiDiagrams/)
38. sphere_cvt, accessed May 17, 2025, [https://people.sc.fsu.edu/~jburkardt/m_src/sphere_cvt/sphere_cvt.html](https://people.sc.fsu.edu/~jburkardt/m_src/sphere_cvt/sphere_cvt.html)
39. people.sc.fsu.edu, accessed May 17, 2025, [https://people.sc.fsu.edu/~jburkardt/m_src/sphere_cvt/sphere_cvt.html#:~:text=The%20CVT%20approximation%20algorithm%20used,overwrite%20XYZ%20with%20this%20data.](https://people.sc.fsu.edu/~jburkardt/m_src/sphere_cvt/sphere_cvt.html#:~:text=The%20CVT%20approximation%20algorithm%20used,overwrite%20XYZ%20with%20this%20data.)
40. Around The World, Part 2: Plate tectonics - Frozen Fractal, accessed May 17, 2025, [https://frozenfractal.com/blog/2023/11/13/around-the-world-2-plate-tectonics/](https://frozenfractal.com/blog/2023/11/13/around-the-world-2-plate-tectonics/)
41. First iteration of my tectonic plate simulation on a sphere (voronoi cells, soft body physics, and Kriging to sample heights at voronoi centroids instead of simulating every pixel) : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/1jhiznp/first_iteration_of_my_tectonic_plate_simulation/](https://www.reddit.com/r/proceduralgeneration/comments/1jhiznp/first_iteration_of_my_tectonic_plate_simulation/)
42. Region Growing, accessed May 17, 2025, [https://homepages.inf.ed.ac.uk/rbf/BOOKS/BANDB/LIB/bandb5.pdf](https://homepages.inf.ed.ac.uk/rbf/BOOKS/BANDB/LIB/bandb5.pdf)
43. Image Display and Basic Processing > Image Processing Tools > Segmentation, accessed May 17, 2025, [https://doc.pmod.com/pbas/pbas_external_pmsegmentationtool.html](https://doc.pmod.com/pbas/pbas_external_pmsegmentationtool.html)
44. Voronoi diagrams--a survey of a fundamental geometric data structure - Johns Hopkins Computer Science, accessed May 17, 2025, [https://www.cs.jhu.edu/~misha/Spring16/Aurenhammer91.pdf](https://www.cs.jhu.edu/~misha/Spring16/Aurenhammer91.pdf)
45. THE ECOLOGY OF GANG TERRITORIAL BOUNDARIES - of Martin Short, accessed May 17, 2025, [https://mshort9.math.gatech.edu/papers/gang_ecology.pdf](https://mshort9.math.gatech.edu/papers/gang_ecology.pdf)
46. Connectivity Recovery Based on Boundary Nodes and Spatial Triangle Fermat Points for Three-Dimensional Wireless Sensor Networks - PMC, accessed May 17, 2025, [https://pmc.ncbi.nlm.nih.gov/articles/PMC11679557/](https://pmc.ncbi.nlm.nih.gov/articles/PMC11679557/)
47. An Overview of Minimum Convex Cover and Maximum Hidden Set - arXiv, accessed May 17, 2025, [https://arxiv.org/html/2403.01354v1](https://arxiv.org/html/2403.01354v1)
48. Efficient generation of simple polygons for characterizing the shape of a set of points in the plane - COSIT, accessed May 17, 2025, [https://geosensor.net/papers/duckham08.PR.pdf](https://geosensor.net/papers/duckham08.PR.pdf)
49. Convex Polygon Packing Based Meshing Algorithm for Modeling of Rock and Porous Media, accessed May 17, 2025, [https://pmc.ncbi.nlm.nih.gov/articles/PMC7302853/](https://pmc.ncbi.nlm.nih.gov/articles/PMC7302853/)
50. Convex hull algorithms - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Convex_hull_algorithms](https://en.wikipedia.org/wiki/Convex_hull_algorithms)
51. 469. Convex Polygon - In-Depth Explanation - AlgoMonster, accessed May 17, 2025, [https://algo.monster/liteproblems/469](https://algo.monster/liteproblems/469)
52. Designing algorithms intuitively : convex decomposition of simple polygons, accessed May 17, 2025, [https://gamedevnotesblog.wordpress.com/2017/10/29/designing-algorithms-intuitively-convex-decomposition-of-simple-polygons/](https://gamedevnotesblog.wordpress.com/2017/10/29/designing-algorithms-intuitively-convex-decomposition-of-simple-polygons/)
53. On Diverse Solutions to Packing and Covering Problems - arXiv, accessed May 17, 2025, [https://www.arxiv.org/pdf/2501.12261](https://www.arxiv.org/pdf/2501.12261)
54. Refolding Planar Polygons - UC Berkeley EECS, accessed May 17, 2025, [https://www2.eecs.berkeley.edu/bears/2004/STARS/final/iben.pdf](https://www2.eecs.berkeley.edu/bears/2004/STARS/final/iben.pdf)
55. Survey of Procedural Methods for Two-Dimensional Texture Generation - PMC, accessed May 17, 2025, [https://pmc.ncbi.nlm.nih.gov/articles/PMC7070409/](https://pmc.ncbi.nlm.nih.gov/articles/PMC7070409/)
56. Efficient generation of simple polygons for characterizing the shape of a set of points in the plane | Request PDF - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/220600217_Efficient_generation_of_simple_polygons_for_characterizing_the_shape_of_a_set_of_points_in_the_plane](https://www.researchgate.net/publication/220600217_Efficient_generation_of_simple_polygons_for_characterizing_the_shape_of_a_set_of_points_in_the_plane)
57. Polygon Decomposition, accessed May 17, 2025, [https://mpen.ca/406/files/PolygonDecomp-Keil.pdf](https://mpen.ca/406/files/PolygonDecomp-Keil.pdf)
58. Chaikin's corner cutting algorithm — smooth_chaikin • smoothr - Matt Strimas-Mackey, accessed May 17, 2025, [https://strimas.com/smoothr/reference/smooth_chaikin.html](https://strimas.com/smoothr/reference/smooth_chaikin.html)
59. Chaikin Curves in Processing · Sighack, accessed May 17, 2025, [https://sighack.com/post/chaikin-curves](https://sighack.com/post/chaikin-curves)
60. cran.r-project.org, accessed May 17, 2025, [https://cran.r-project.org/web/packages/smoothr/vignettes/smoothr.html#:~:text=Chaikin's%20corner%20cutting%20algorithm%20smooths,way%20to%20the%20previous%20point.](https://cran.r-project.org/web/packages/smoothr/vignettes/smoothr.html#:~:text=Chaikin's%20corner%20cutting%20algorithm%20smooths,way%20to%20the%20previous%20point.)
61. It's the simple stuff that matters - SketchUp Blog, accessed May 17, 2025, [https://blog.sketchup.com/article/its-the-simple-stuff-that-matters](https://blog.sketchup.com/article/its-the-simple-stuff-that-matters)
62. vtkSmoothPolyDataFilter Class Reference - VTK, accessed May 17, 2025, [https://vtk.org/doc/release/5.4/html/a01566.html](https://vtk.org/doc/release/5.4/html/a01566.html)
63. Iterative Smoothing of Curves and Surfaces, accessed May 17, 2025, [https://airccse.org/journal/ijcga/papers/3113ijcga03.pdf](https://airccse.org/journal/ijcga/papers/3113ijcga03.pdf)
64. Maya Help | Smooth a mesh by averaging the distance between vertices | Autodesk, accessed May 17, 2025, [https://help.autodesk.com/view/MAYAUL/2025/ENU/?guid=GUID-7DA6670A-625B-4734-864F-16821910D062](https://help.autodesk.com/view/MAYAUL/2025/ENU/?guid=GUID-7DA6670A-625B-4734-864F-16821910D062)
65. How Smooth Line and Smooth Polygon work—ArcGIS Pro | Documentation, accessed May 17, 2025, [https://pro.arcgis.com/en/pro-app/3.3/tool-reference/cartography/how-smooth-line-and-smooth-polygon-work.htm](https://pro.arcgis.com/en/pro-app/3.3/tool-reference/cartography/how-smooth-line-and-smooth-polygon-work.htm)
66. Douglas-Peucker algorithm | Cartography Playground, accessed May 17, 2025, [https://cartography-playground.gitlab.io/playgrounds/douglas-peucker-algorithm/](https://cartography-playground.gitlab.io/playgrounds/douglas-peucker-algorithm/)
67. How Simplify Line and Simplify Polygon work - ArcMap Resources for ArcGIS Desktop, accessed May 17, 2025, [https://desktop.arcgis.com/en/arcmap/latest/tools/cartography-toolbox/how-simplify-line-works.htm](https://desktop.arcgis.com/en/arcmap/latest/tools/cartography-toolbox/how-simplify-line-works.htm)
68. Polygon Detection from a Set of Lines - arXiv, accessed May 17, 2025, [https://arxiv.org/html/2312.16363v1](https://arxiv.org/html/2312.16363v1)
69. Polygon Detection from a Set of Lines, accessed May 17, 2025, [https://web.tecnico.ulisboa.pt/alfredo.ferreira/publications/12EPCG-PolygonDetection.pdf](https://web.tecnico.ulisboa.pt/alfredo.ferreira/publications/12EPCG-PolygonDetection.pdf)
70. Simplify By Straight Lines And Circular Arcs (Editing)—ArcGIS Pro | Documentation, accessed May 17, 2025, [https://pro.arcgis.com/en/pro-app/latest/tool-reference/editing/simplifybystraightlinesandcirculararcs.htm](https://pro.arcgis.com/en/pro-app/latest/tool-reference/editing/simplifybystraightlinesandcirculararcs.htm)
71. Circular approximation of polygon (or its part) - Stack Overflow, accessed May 17, 2025, [https://stackoverflow.com/questions/27250353/circular-approximation-of-polygon-or-its-part](https://stackoverflow.com/questions/27250353/circular-approximation-of-polygon-or-its-part)
72. Curves in 3D - 3D Math Primer for Graphics and Game Development, accessed May 17, 2025, [https://gamemath.com/book/curves.html](https://gamemath.com/book/curves.html)
73. Implementing Procedural Generation - GDevelop documentation, accessed May 17, 2025, [https://wiki.gdevelop.io/gdevelop5/tutorials/procedural-generation/implementing-procedural-generation/](https://wiki.gdevelop.io/gdevelop5/tutorials/procedural-generation/implementing-procedural-generation/)
74. Procedural generation : r/roguelikedev - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/roguelikedev/comments/1f7s4e3/procedural_generation/](https://www.reddit.com/r/roguelikedev/comments/1f7s4e3/procedural_generation/)
75. Lithospheric Buoyancy and Continental Intraplate Stresses - Earthquake Science Center, accessed May 17, 2025, [https://escweb.wr.usgs.gov/share/mooney/2003_IGR_LithoBuoyancy.pdf](https://escweb.wr.usgs.gov/share/mooney/2003_IGR_LithoBuoyancy.pdf)
76. Ocean ridge system | EBSCO Research Starters, accessed May 17, 2025, [https://www.ebsco.com/research-starters/earth-and-atmospheric-sciences/ocean-ridge-system](https://www.ebsco.com/research-starters/earth-and-atmospheric-sciences/ocean-ridge-system)
77. Digital Elevation Models: Terminology and Definitions - MDPI, accessed May 17, 2025, [https://www.mdpi.com/2072-4292/13/18/3581](https://www.mdpi.com/2072-4292/13/18/3581)
78. Heightmaps and Voronoi Diagrams: Revolutionizing Game World Generation - Wayline, accessed May 17, 2025, [https://www.wayline.io/blog/heightmaps-voronoi-diagrams-game-world-generation](https://www.wayline.io/blog/heightmaps-voronoi-diagrams-game-world-generation)
79. Voronoi Diagrams: Understanding the basic technique for breakable geometry, path finding, random planet generation, etc... - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/gamedev/comments/47c9jz/voronoi_diagrams_understanding_the_basic/](https://www.reddit.com/r/gamedev/comments/47c9jz/voronoi_diagrams_understanding_the_basic/)