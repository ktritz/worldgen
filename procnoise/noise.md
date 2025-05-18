<!-----



Conversion time: 4.374 seconds.


Using this Markdown file:

1. Paste this output into your source file.
2. See the notes and action items below regarding this conversion run.
3. Check the rendered output (headings, lists, code blocks, tables) for proper
   formatting and use a linkchecker before you publish this page.

Conversion notes:

* Docs to Markdown version 1.0β44
* Sat May 17 2025 16:05:21 GMT-0700 (PDT)
* Source doc: Start research
* Tables are currently converted to HTML tables.
----->



# Advanced Noise Algorithms for Comprehensive Procedural World Generation


## 1. Introduction


### The Indispensable Role of Noise in Procedural World Generation

Procedural generation is a cornerstone of modern digital content creation, particularly in the development of expansive and intricate virtual worlds. This methodology facilitates the algorithmic creation of content, significantly reducing the reliance on manual authoring and enabling the emergence of complex, varied, and often unpredictable environments.<sup>1</sup> At the heart of most procedural generation systems lie noise functions. These mathematical constructs are not "noise" in the disruptive sense, but rather a source of controlled, pseudo-random data that can be sculpted to emulate the complex and often chaotic patterns found in nature.<sup>3</sup> Noise functions are pivotal in generating a vast array of elements, including textures, three-dimensional models, terrain features, atmospheric effects, and even influencing the distribution of in-world resources or the behavior of simulated entities.<sup>2</sup>

The selection and application of noise algorithms extend beyond mere aesthetic considerations. The choice profoundly impacts computational performance, the verisimilitude of simulated natural systems, and the ability to generate specific geological, climatic, and ecological phenomena. For a project aiming to model a world with diverse interacting systems—such as tectonic plates, global climate patterns, and detailed erosion features—a simplistic, monolithic approach to noise is inadequate. Different natural processes exhibit distinct characteristic patterns: the smooth, rolling undulations of ancient hills differ vastly from the sharp, cellular structure of drying mud or the turbulent, swirling patterns of atmospheric currents. Similarly, different noise algorithms excel at producing these varied patterns. For instance, Perlin noise is renowned for its smooth gradients suitable for organic terrain <sup>3</sup>, Worley noise generates cellular structures ideal for certain biological or crystalline patterns <sup>9</sup>, and Curl noise can produce divergence-free vector fields essential for fluid simulations.<sup>10</sup> Consequently, a sophisticated procedural world model demands a diverse and carefully curated toolkit of noise functions, where each algorithm is chosen and tuned for its specific applicability to the subsystem it aims to represent. The ultimate realism and complexity of the generated world will arise not from a single noise function, but from the nuanced layering, combination, and modulation of multiple noise sources, allowing for the emergence of intricate details and believable systemic interactions.


### User Objective and Report Scope

This report addresses the objective of identifying and evaluating additional noise algorithms that can augment a comprehensive procedural world generation project. The project currently includes a noise module and aims to model tectonic plates, elevation, climate (encompassing energy balance, air and water circulation patterns, rainfall, and biomes), resource distribution, erosion processes, and watershed delineation. The scope of this investigation includes a thorough examination of established noise algorithms employed in existing world generation systems, as well as an exploration of novel or less commonly utilized algorithms. For each pertinent algorithm identified, this report will detail its underlying principles, assess its potential utility for the specific elements of the user's project, and identify existing code libraries, with a preference for implementations in Go or other portable languages.


## 

---
2. Foundational Noise Algorithms in World Generation

A number of noise algorithms have become foundational in the field of procedural generation due to their versatility and computational properties. Understanding their core principles is essential before exploring more advanced or specialized techniques.


### 2.1. Perlin Noise

Developed by Ken Perlin in 1983, Perlin noise is a type of gradient noise that has become a mainstay in computer graphics for generating natural-looking textures and phenomena.<sup>3</sup> Its mathematical basis involves generating pseudo-random gradient vectors at the corners of a grid and then interpolating these gradients to determine the noise value at any point within the grid.<sup>3</sup> This process, involving dot products and smooth interpolation functions (often a 3rd or 5th degree polynomial to ensure continuous derivatives), results in smooth, continuous patterns that mimic many natural forms.<sup>3</sup>

**Strengths:** Perlin noise is widely understood and relatively straightforward to implement for basic versions. It is computationally efficient for generating smooth, organic-looking outputs, making it suitable for base terrain heightmaps, textures like clouds or water, and smooth value distributions for parameters like temperature or rainfall.<sup>3</sup>

**Weaknesses/Artifacts:** A well-known limitation of classic Perlin noise is its tendency to produce visible artifacts aligned with the underlying grid, especially in 2D implementations or when scaled over large areas.<sup>3</sup> These artifacts can manifest as features that are more prominent along axial or diagonal directions. While techniques exist to mitigate this, it remains a consideration. For many applications, particularly those requiring higher dimensional noise or fewer artifacts, Simplex noise is often preferred.<sup>2</sup>

**Relevance to Project Elements:** Perlin noise is commonly used for generating base elevation maps <sup>14</sup>, initial temperature and rainfall distributions <sup>3</sup>, cloud patterns <sup>3</sup>, and as a component in biome definition.<sup>3</sup> In the context of tectonic plate generation, it has been used to distort grid coordinates to create more organic plate shapes or to define initial plate height variations.<sup>14</sup>


### 2.2. Simplex Noise

Also developed by Ken Perlin, in 2001, Simplex noise was designed as an improvement over his classic Perlin noise, particularly to address its computational complexity in higher dimensions and its directional artifacts.<sup>18</sup> Instead of a hypercubic grid, Simplex noise operates on a simplicial grid (e.g., triangles in 2D, tetrahedra in 3D). This fundamentally reduces the number of corner points that need to be evaluated to calculate the noise value at a given point; an n-dimensional simplex has n+1 vertices, whereas an n-dimensional hypercube has 2n vertices.<sup>18</sup>

**Advantages over Perlin Noise:**



* **Lower Computational Complexity:** Simplex noise scales more favorably to higher dimensions, with a complexity of O(n2) compared to Perlin noise's O(n⋅2n).<sup>18</sup> This makes it significantly faster for 3D, 4D, or higher-dimensional noise.
* **Fewer Directional Artifacts:** The use of a simplicial grid and a different gradient distribution scheme results in noise that is more visually isotropic, lacking the prominent grid-aligned artifacts often seen with Perlin noise.<sup>18</sup>
* **Well-Defined Continuous Gradient:** Simplex noise typically has a well-defined and continuous gradient (almost everywhere), which is beneficial for applications requiring surface normals or other gradient-based calculations.<sup>18</sup>

**Relevance to Project Elements:** Due to its advantages, Simplex noise is often preferred over Perlin noise for many procedural generation tasks, including terrain elevation <sup>6</sup>, fluid dynamics simulations (e.g., for cloud or water animation <sup>3</sup>), and complex texture generation.<sup>5</sup> Its improved isotropy is particularly beneficial for large-scale features like continental landmasses or broad climatic zones, where grid artifacts from Perlin noise could become very noticeable and detract from realism.


### 2.3. Value Noise

Value noise is conceptually simpler than gradient noises like Perlin or Simplex noise. It operates by assigning pseudo-random scalar values directly to the vertices of an integer lattice.<sup>11</sup> The noise value at any point within a lattice cell is then determined by interpolating the values from the surrounding cell vertices.<sup>20</sup> Common interpolation methods include linear, bilinear/trilinear, or cubic/bicubic/tricubic interpolation, with higher-order methods producing smoother results at a higher computational cost.<sup>21</sup>

**Comparison to Gradient Noise:** The primary difference lies in what is stored at the lattice points: random scalar values for value noise versus random gradient vectors for gradient noise.<sup>11</sup> This often leads to value noise appearing somewhat "blockier" or less "organic" than gradient noise, especially with linear interpolation, though this can be desirable for certain effects.<sup>21</sup>

**Relevance to Project Elements:** While perhaps less suited for generating the primary smooth, rolling terrain compared to Perlin or Simplex noise, value noise can be useful for:



* Creating simpler, more regular patterns.
* Modulating other noise functions or procedural systems. For example, it has been used to control the distortion of grid corners in tectonic plate generation to break up regularity.<sup>14</sup>
* Generating blocky or terraced features if desired.
* Serving as a base for fBm when a more rugged or less "gradient-smooth" fractal character is intended. The inherent nature of value noise can lead to fBm with more distinct plateaus or sharper transitions between levels, which might be suitable for certain types of stratified rock or stylized terrain.


### 2.4. Worley Noise (Cellular Noise)

Introduced by Steven Worley in 1996, Worley noise (also known as Cellular noise or Voronoi noise) generates patterns based on the distance to a set of randomly distributed feature points, or "seeds".<sup>9</sup> For any given sample point, the noise value is typically derived from the distance to the n-th nearest feature point.

**Principles and Variations:**



* **Seed Distribution:** Feature points are scattered throughout space, often on a grid for efficient lookup, though their exact positions within each grid cell are usually jittered randomly.<sup>9</sup>
* **Distance Calculation:** Common distance metrics include Euclidean, Manhattan, or Chebyshev.<sup>22</sup>
* **Fn​ Values:** The core outputs are distances F1​,F2​,F3​,…, representing the distance to the 1st, 2nd, 3rd, etc., nearest feature point.<sup>9</sup>
    * **F1​:** Generates classic Voronoi cells, where each region is closest to a single seed point. This is useful for defining distinct zones or regions.<sup>9</sup>
    * **F2​:** Distance to the second-closest point. Can create more complex patterns, often highlighting areas equidistant from two points.<sup>9</sup>
    * **F2​−F1​:** This common variation emphasizes the boundaries between Voronoi cells, creating vein-like or crack-like patterns.<sup>25</sup>
    * Other combinations (e.g., F1​+F2​, F1​×F2​, F2​/F1​) can produce a wide variety of effects.<sup>22</sup> Some libraries also provide access to the ID or value associated with the feature points.

**Relevance to Project Elements:**



* **Tectonic Plates:** Can define the initial shapes of tectonic plates or the regions of influence for different plate characteristics.<sup>17</sup> The F2​−F1​ variation is particularly interesting for defining initial fault lines or rifts along plate boundaries.
* **Resource Distribution:** Useful for creating clustered resource deposits, where resources might appear within specific cellular regions or along their boundaries.<sup>5</sup>
* **Biome Definition:** Can delineate distinct biome regions based on proximity to biome "seed" points.<sup>29</sup>
* **Erosion Patterns:** Can simulate cracked earth or certain types of rock weathering patterns.<sup>5</sup>
* **Material Properties:** Can define regions of varying rock hardness or other material properties that influence erosion. The cellular nature of Worley noise makes it ideal for systems where distinct, somewhat irregularly shaped regions are needed. A hierarchical application, where F1​ defines broader regions (like plate interiors or biome cores) and F2​−F1​ defines features along their interfaces (like fault lines or ecotones), is a powerful approach.


### 2.5. Fractal Brownian Motion (fBm)

Fractal Brownian Motion, often referred to as fractal noise, is not a distinct noise algorithm itself but rather a technique for combining multiple instances (octaves) of a base noise function (such as Perlin, Simplex, or Value noise) to create more detailed and self-similar patterns.<sup>31</sup>

Construction and Parameters:

fBm is constructed by summing several layers of noise. Each successive layer, or "octave," typically has:



* **Increased Frequency:** The frequency of the noise is multiplied by a factor called *lacunarity* (commonly 2.0), meaning each subsequent octave adds finer details.<sup>31</sup>
* **Decreased Amplitude:** The amplitude of the noise is multiplied by a factor called *gain* or *persistence* (commonly 0.5), meaning each subsequent octave contributes less to the overall height, making the finer details less prominent than the larger forms.<sup>31</sup> The number of *octaves* determines the overall level of detail; more octaves lead to richer textures but increase computation time.<sup>31</sup>

**Relevance to Project Elements:** fBm is fundamental for most forms of naturalistic terrain generation.



* **Elevation:** Low-frequency octaves create large-scale features like continents, mountain ranges, and broad valleys, while high-frequency octaves add smaller details like hills, ridges, and surface roughness.<sup>7</sup>
* **Climate:** Generating cloud patterns with varying levels of detail, or creating turbulent water surfaces.<sup>3</sup>
* **Textures:** Adding detail and realism to virtually any procedurally generated texture. The choice of the underlying base noise (Perlin, Simplex, or Value) significantly influences the character of the resulting fBm. For instance, fBm built upon Value noise might produce more terraced or angular fractal patterns compared to the smoother, more organic results typically obtained from Perlin or Simplex-based fBm. This allows for stylistic control over the generated features.

The following table provides a summary of these foundational noise algorithms.

**Table 1: Overview of Foundational Noise Algorithms**


<table>
  <tr>
   <td><strong>Algorithm</strong>
   </td>
   <td><strong>Core Principle</strong>
   </td>
   <td><strong>Key Strengths</strong>
   </td>
   <td><strong>Common Artifacts/Weaknesses</strong>
   </td>
   <td><strong>Primary Use Cases in World Gen (Examples)</strong>
   </td>
  </tr>
  <tr>
   <td><strong>Perlin Noise</strong>
   </td>
   <td>Interpolated pseudo-random gradients on a grid <sup>3</sup>
   </td>
   <td>Computationally efficient (basic), smooth, organic patterns <sup>3</sup>
   </td>
   <td>Grid-aligned artifacts, especially in 2D or higher dimensions <sup>3</sup>
   </td>
   <td>Base elevation maps <sup>14</sup>, temperature/rainfall maps <sup>3</sup>, cloud patterns <sup>3</sup>, biome definition <sup>3</sup>
   </td>
  </tr>
  <tr>
   <td><strong>Simplex Noise</strong>
   </td>
   <td>Interpolated pseudo-random gradients on a simplicial grid <sup>18</sup>
   </td>
   <td>Lower complexity (esp. high-dim), fewer artifacts, isotropic <sup>18</sup>
   </td>
   <td>More complex to implement than Perlin <sup>6</sup>
   </td>
   <td>Preferred for elevation <sup>6</sup>, fluid dynamics <sup>3</sup>, complex textures <sup>5</sup>; generally where Perlin artifacts are an issue.
   </td>
  </tr>
  <tr>
   <td><strong>Value Noise</strong>
   </td>
   <td>Interpolated pseudo-random values on a grid <sup>20</sup>
   </td>
   <td>Conceptually simple, can be fast with simple interpolation <sup>21</sup>
   </td>
   <td>Can appear blocky or less organic than gradient noise <sup>21</sup>
   </td>
   <td>Simpler patterns, modulating other noises (e.g., tectonic grid distortion <sup>14</sup>), base for fBm if blockier fractals desired.
   </td>
  </tr>
  <tr>
   <td><strong>Worley Noise</strong>
   </td>
   <td>Distance to n-th nearest feature points (seeds) <sup>9</sup>
   </td>
   <td>Generates cellular/Voronoi patterns, good for distinct regions <sup>9</sup>
   </td>
   <td>Can look artificial if used alone for terrain <sup>6</sup>
   </td>
   <td>Tectonic plate outlines <sup>27</sup>, resource clustering, biome region definition <sup>29</sup>, cracked earth patterns.<sup>5</sup> F2​−F1​ for veins/boundaries.
   </td>
  </tr>
  <tr>
   <td><strong>fBm</strong>
   </td>
   <td>Summing octaves of a base noise with varying frequency/amplitude <sup>32</sup>
   </td>
   <td>Creates detailed, self-similar fractal patterns <sup>32</sup>
   </td>
   <td>Can be computationally expensive with many octaves <sup>31</sup>
   </td>
   <td>Essential for terrain heightmaps (mountains, valleys <sup>32</sup>), clouds <sup>32</sup>, water surfaces <sup>31</sup>; adding detail to any noise-driven system.
   </td>
  </tr>
</table>



## 

---
3. Advanced and Novel Noise Algorithms for Enhanced World Modeling

Beyond the foundational algorithms, a range of more specialized or newer noise techniques offer unique capabilities that can significantly enhance the realism and complexity of procedurally generated worlds.


### 3.1. Gabor Noise

Gabor noise is a type of sparse convolution noise that utilizes Gabor kernels—essentially a Gaussian function modulated by a sinusoidal wave.<sup>35</sup> This construction provides excellent control over the spectral properties of the noise, particularly its frequency content and orientation.<sup>37</sup>

**Characteristics:**



* **Directionality and Anisotropy:** One of the defining features of Gabor noise is its ability to produce strongly anisotropic patterns, resembling brush strokes, wood grain, or muscle fibers.<sup>37</sup> Parameters such as 'Orientation' allow for explicit control over the direction of these features.
* **Frequency and Bandwidth Control:** Gabor noise allows for direct manipulation of feature density via a 'Frequency' parameter and smoothness/randomness through a 'Bandwidth' parameter.<sup>37</sup>
* **Paintable Flow:** Some advanced implementations permit the directional properties of Gabor noise to be guided by user-defined vector fields, effectively allowing artists to "paint" the flow or orientation of the noise patterns.<sup>40</sup>

Potential Project Applications:

The anisotropic nature of Gabor noise makes it highly suitable for generating geological and biological features that exhibit strong directionality.



* **Tectonic Plates and Elevation:** It can be employed to create large-scale anisotropic terrain features such as elongated sand dunes (sef dunes), yardangs (wind-eroded ridges), or patterns indicative of glacial scouring, where directional forces have shaped the landscape.<sup>40</sup> It could also simulate striations on rock faces or represent directional stress patterns within geological formations.
* **Resource Distribution:** Gabor noise is well-suited for modeling anisotropic resource deposits, such as layered mineral veins that follow specific geological orientations, or fibrous organic resources.<sup>41</sup>
* **Biomes and Textures:** It can generate realistic textures for specific biome elements, such as wood grain in trees <sup>37</sup>, muscle fibers in creatures, or patterns of wind-swept grasses and vegetation. The "sparse" characteristic of Gabor noise, resulting from the convolution of sparsely distributed impulses <sup>35</sup>, differentiates its computational profile from lattice-based noises. Performance for large-scale generation will depend on impulse density and kernel radius, requiring careful tuning.


### 3.2. Wavelet Noise

Wavelet noise, notably the method developed at Pixar by Cook and DeRose, is classified as an "explicit noise".<sup>35</sup> Its generation involves creating an initial random noise image (or tile), then processing it through downsampling and upsampling stages, and finally subtracting the processed version from the original.<sup>8</sup> This procedure effectively isolates band-limited noise components.

**Benefits:**



* **Band-Limiting:** The primary advantage of Wavelet noise is its nearly perfect band-limited nature.<sup>35</sup> This significantly reduces aliasing artifacts and loss of detail, which can be problematic with other noise functions like Perlin noise, especially when mapping 3D noise onto 2D surfaces or when dealing with high frequencies near the Nyquist limit.
* **Spectral Control:** The band-limited components can be combined (similar to octaves in fBm) with specific weights to achieve precise control over the resulting noise's power spectrum.<sup>35</sup>

Potential Project Applications:

The distinct separation of frequency bands makes Wavelet noise particularly useful for creating features with clear layering or stratification.



* **Elevation and Terrain:** Ideal for generating terrain with distinct geological strata, where different frequency bands correspond to different rock layers possessing varied visual characteristics or erodibility.<sup>47</sup> This can be foundational for realistic erosion simulation.
* **Climate Modeling:** Creating layered cloud formations where different altitudes exhibit distinct cloud types or densities, or modeling stratified atmospheric phenomena.<sup>8</sup>
* **Resource Distribution:** Defining stratified resource deposits, such as sedimentary mineral layers. The "explicit" nature of Wavelet noise, often involving pre-generated tiles, implies an architectural consideration for very large or infinite worlds; strategies for tiling or dynamically generating these noise bands for local regions may be necessary.


### 3.3. Curl Noise

Curl noise, introduced by Bridson et al. for procedural fluid flow, is generated by taking the curl (∇×) of a vector potential field.<sup>10</sup> This potential field is often itself derived from another noise function, such as 3D Perlin or Simplex noise, where different components of the potential (ψx, ψy, ψz) are sampled from the base noise, possibly with offsets.<sup>10</sup>

**Characteristics:**



* **Divergence-Free Fields:** The most significant property of Curl noise is that the resulting vector field is mathematically guaranteed to be divergence-free (∇·v = 0).<sup>10</sup> This means the flow it represents is incompressible, with no sources or sinks, which is characteristic of many natural fluid motions.
* **Turbulent/Swirling Patterns:** Curl noise naturally produces swirling, vortex-like, and turbulent patterns, ideal for fluid-like effects.<sup>55</sup>

Potential Project Applications:

Its ability to generate incompressible, turbulent vector fields makes Curl noise highly valuable for physical simulations within the world.



* **Air and Water Circulation:** Curl noise is exceptionally well-suited for generating large-scale and fine-detail vector fields for wind <sup>12</sup> and ocean currents, including gyres and eddies.<sup>56</sup> This can drive the transport of heat and moisture in the climate model.
* **Climate Modeling:** Simulating dynamic weather patterns, the movement of smoke plumes, volcanic ash, or other atmospheric particle effects.<sup>55</sup> It can contribute to the appearance of persistent high/low-pressure cells by defining the swirling air motion around them.
* **Erosion Simulation:** The vector fields generated by Curl noise can guide the paths of particles in hydraulic erosion simulations, leading to more natural and complex channel formations and deposition patterns. Effective implementation often requires access to the gradients of the underlying potential noise. If analytical derivatives are not available, finite differences can be used, though this may impact performance and accuracy.<sup>10</sup>


### 3.4. Flow Noise

Flow noise is primarily an animation technique designed to create dynamic, flowing patterns that appear more natural than simple translation of static noise.<sup>35</sup> It often involves time-varying modifications to a base noise function, typically Simplex noise due to its quality and efficiency.

**Principles:**



* **Gradient Rotation:** A key aspect is the rotation of the generating gradients of the base noise function over time. For 2D noise, this might involve adding a time-dependent angle to the pseudo-random angle used to define gradients. For 3D, gradients can be rotated around a pseudo-random axis by a time-dependent angle.<sup>59</sup>
* **Domain Advection:** The input coordinates to the noise function can be advected (warped) over time, often in the direction of the noise gradient itself (or its perpendicular for swirling), to enhance the appearance of flow and turbulence.<sup>59</sup>
* **Tiling:** Implementations, such as the one detailed in the JCGT paper by Gustavson et al., can be designed to produce tiling flow noise, suitable for looping animations or seamless textures.<sup>59</sup>

**Potential Project Applications:**



* **Air and Water Circulation:** Animating the dynamic evolution of wind patterns, ocean currents, and river flow over time, providing a visually active world.
* **Climate Modeling:** Visualizing shifting weather systems, evolving cloud formations, the movement of temperature fronts, or the dispersal of atmospheric particles.
* **Erosion Simulation:** Simulating the time-varying flow of water or wind that drives erosion processes, potentially showing changes in erosion intensity or direction.
* **Animated Textures:** Creating dynamic textures for water surfaces, flowing lava, or atmospheric effects.


### 3.5. Anisotropic Noise

Anisotropic noise refers to noise patterns whose statistical properties vary with direction, contrasting with isotropic noises (like ideal Perlin or Simplex) which appear uniform regardless of rotation.<sup>8</sup> This directionality is crucial for representing many natural and man-made phenomena.

**Generation Techniques:**



* **Steerable Filters:** Applying filters that can be oriented in specific directions to an isotropic noise source or to white noise.<sup>8</sup>
* **Frequency Domain Manipulation:** Decomposing a noise signal into oriented sub-bands in the frequency domain and then reconstructing it.<sup>61</sup>
* **Specialized Noise Functions:** Algorithms like Gabor noise (see Section 3.1) are inherently anisotropic due to their kernel definitions.
* **Coordinate Scaling:** Non-uniform scaling of input coordinates to an isotropic noise function can introduce simple anisotropy, though often with less control than dedicated methods.

**Potential Project Applications:**



* **Elevation and Terrain:** Creating geological features shaped by directional forces, such as:
    * Wind-swept rock formations and sand dunes (e.g., barchan or seif dunes).<sup>40</sup>
    * Yardangs (elongated, wind-eroded ridges).
    * Glacial scouring marks, drumlins, and eskers.
    * Striated rock textures or sedimentary layers with a preferred orientation.
* **Resource Distribution:** Modeling mineral veins or other resource deposits that tend to align along geological faults or stress fields.
* **Biomes and Textures:** Generating textures for materials with inherent directionality, such as wood grain <sup>40</sup>, muscle fibers, brushed metals, or aligned grasses in a windswept plain.


### 3.6. Domain Warping / Distortion

Domain warping, also known as domain distortion or turbulence, is a powerful meta-technique that involves altering the input coordinates (the domain) of a primary noise function using the output of one or more secondary noise functions.<sup>64</sup> This "warps" the space in which the primary noise is evaluated, leading to more complex and often more organic-looking patterns.

**Characteristics:**



* **Increased Complexity:** Introduces swirls, eddies, and non-linear distortions into the base noise pattern.
* **Organic Appearance:** Can break up the inherent regularity or grid-like tendencies of some noise functions, yielding more naturalistic results.
* **Versatility:** Can be applied to virtually any base noise function (Perlin, Simplex, Worley, etc.) and in any number of dimensions.

**Potential Project Applications:**



* **Elevation and Terrain:**
    * Simulating the appearance of tectonically deformed terrain, such as folded mountain ranges or complex fault zones.<sup>64</sup>
    * Creating more varied and less uniform mountain shapes, hills, and valleys.
    * Generating intricate river meanders or complex coastlines by warping the domain of the elevation noise or a water mask.
* **Climate Modeling:**
    * Enhancing the visual complexity of cloud patterns, making them appear more turbulent and less like simple fBm.<sup>66</sup>
    * Creating more realistic water surface animations with interacting wave patterns.
    * Simulating turbulent atmospheric phenomena.
* **Resource Distribution:** Generating more irregular and natural-looking pathways for ore veins or less uniform clusters of resources.
* **Biome Definition:** Adding natural irregularity to biome boundaries or creating more complex internal patterns within biomes. The game *No Man's Sky* famously uses domain warping in its "uber noise" function to generate diverse planetary terrains.<sup>64</sup>


### 3.7. Multi-Fractal Noise

Standard Fractal Brownian Motion (fBm) uses constant scaling factors (lacunarity and gain) across all octaves. Multi-fractal noise extends this concept by allowing these scaling parameters to vary, or by combining octaves in more sophisticated ways than simple addition.<sup>25</sup> This results in a noise field where the fractal dimension itself can vary spatially, leading to more heterogeneous and often more realistic natural patterns.

**Types and Characteristics:**



* **Heterogeneous fBm:** Lacunarity or gain can change from one octave to the next, or even spatially based on another noise function.
* **Ridged Multifractal:** Often involves taking the absolute value of the noise at each octave and then inverting it (e.g., 1.0−∣noise(P)∣) before accumulation. This creates sharp ridges and valleys.<sup>25</sup>
* **Billow Noise:** Similar to ridged noise, but often involves offsetting the absolute value to create a "billowy" or puffy appearance, like clouds or turbulent smoke.<sup>75</sup>
* **Hybrid Multifractal:** Combines different octaves or noise types using more complex operations than simple weighted addition, such as multiplication or using the output of one octave to modulate the parameters of another.<sup>33</sup>

**Potential Project Applications:**



* **Elevation and Terrain:** This is a primary application. Multi-fractal noise can generate terrains where some areas are relatively smooth (e.g., plains with a lower effective fractal dimension) while others are extremely rugged and detailed (e.g., young, jagged mountain ranges with a higher effective fractal dimension), all within a single coherent noise system.<sup>25</sup> This is crucial for creating varied and believable landscapes.
* **Climate Modeling:** Modeling complex cloud formations that exhibit both smooth, diffuse areas and sharply defined, turbulent regions.
* **Erosion Simulation:** Defining spatially varying rock hardness or erodibility. A multi-fractal noise field could represent rock that is consistently hard in some areas and highly fractured or variable in others.
* **Resource Distribution:** Creating resource fields where the density, patchiness, and clustering of resources vary significantly across different regions, reflecting diverse geological processes.


### 3.8. Pattern-Based Noise & Hybrid Approaches

This category encompasses techniques where deterministic or rule-based pattern generation methods are combined with noise functions to introduce variation, guide placement, or modulate properties.<sup>1</sup> This allows for the creation of structures that are more ordered or exhibit specific grammars than pure noise, yet still possess natural-looking irregularities.

**Examples of Hybrid Approaches:**



* **L-Systems with Noise Modulation:** Lindenmayer systems (L-systems) are excellent for generating branching structures like plants, river networks, or vein systems.<sup>79</sup> Noise can be introduced to:
    * Control branching angles, segment lengths, or thickness randomly.<sup>89</sup>
    * Determine stochastically which production rule to apply in a stochastic L-system.<sup>82</sup>
    * Perturb the "turtle" position or orientation during geometric interpretation.
    * Modify a base noise field (e.g., Perlin) according to the L-system's output to carve paths or structures.<sup>83</sup>
* **Reaction-Diffusion with Noise:** Reaction-diffusion systems can generate organic patterns like spots or stripes. Noise can be used to initialize the chemical concentrations or perturb the diffusion rates, leading to more varied and less uniform patterns.
* **Tiling and Wang Tiles:** Noise can be used to select which tiles to place or to modulate the appearance of pre-authored tiles to create varied, non-repetitive surfaces from a limited set of assets.
* **Noise-Guided Placement:** Using a noise field to determine the probability or density of placing objects (e.g., trees, rocks, resource nodes) generated by other means.

**Potential Project Applications:**



* **Resource Distribution:** Generating complex, branching ore vein networks using L-systems, where noise controls vein thickness, branching probability, mineral concentration, or meandering.<sup>79</sup>
* **Watersheds and River Networks:** Defining the primary paths of rivers and tributaries using L-systems or other pathfinding algorithms, with noise functions modulating river width, sinuosity, and the surrounding terrain modifications (e.g., carving into a heightmap).
* **Biome Features:** Creating specific patterned vegetation within biomes, or complex root systems.
* **Erosion Features:** Guiding erosion processes along structurally weak lines defined by a pattern, with noise affecting the intensity of erosion. The combination of rule-based generation with noise allows for a higher level of emergent complexity, where structured forms gain organic irregularities, leading to more believable and diverse procedural content.


### 3.9. Spot Noise

The term "Spot Noise" is somewhat ambiguous in the context of general procedural graphics, as it's not as formally defined as Perlin or Worley noise. In some contexts, particularly related to sparse convolution noise like Gabor noise, it can refer to the individual kernels or "spots" that are distributed and summed to create the final noise pattern.<sup>35</sup> Each spot contributes locally to the noise field. Another usage, found in a Reddit thread <sup>71</sup>, mentions "spot noise" in the context of terrain generation but without a clear algorithmic definition, possibly implying localized application of noise or noise with very distinct, isolated features. A different domain, electronic countermeasures, uses "Spot Noise" to refer to narrowband noise energy, which is not relevant to graphical applications.<sup>91</sup>

Interpreted Characteristics for Graphics:

If "Spot Noise" implies a method based on distributing discrete visual elements or localized noise contributions, its characteristics would be:



* **Localized Influence:** Each "spot" affects a limited area.
* **Density Control:** The overall appearance would depend on the density and distribution of these spots.
* **Shape of Spots:** The shape of the individual spots/kernels would define the local texture.

**Potential Project Applications (Interpreted as Localized Feature Noise):**



* **Resource Distribution:** Placing discrete resource nodes (e.g., individual crystals, small patches of rare minerals) or small, distinct clusters.
* **Terrain Detailing:** Adding localized details like individual boulders, small rock outcrops, or patches of specific ground cover.
* **Biome Features:** Placing individual, notable trees, unique plants, or small, distinct terrain features within a larger biome.
* **Erosion/Weathering:** Simulating localized weathering effects like pockmarks on rocks.

Given the ambiguity, if precise control over localized features is needed, techniques like sparse convolution noise (e.g., Gabor noise with controlled impulse density and kernel size) or object placement algorithms driven by a density map (which itself could be noise) would offer more formally defined and controllable approaches than a vaguely defined "Spot Noise."

The following table summarizes these advanced and novel noise algorithms.

**Table 2: Advanced & Novel Noise Algorithms – Properties and Potential**


<table>
  <tr>
   <td><strong>Algorithm</strong>
   </td>
   <td><strong>Core Principle</strong>
   </td>
   <td><strong>Unique Characteristics</strong>
   </td>
   <td><strong>Potential Applications for Project Elements</strong>
   </td>
   <td><strong>Notable Library Support (Brief)</strong>
   </td>
  </tr>
  <tr>
   <td><strong>Gabor Noise</strong>
   </td>
   <td>Sparse convolution with Gabor kernels (Gaussian x sinusoid) <sup>35</sup>
   </td>
   <td>Anisotropic, directional, good spectral control <sup>37</sup>
   </td>
   <td>Anisotropic terrain (dunes, strata), wood grain, muscle fibers, directional resource veins <sup>41</sup>
   </td>
   <td>OpenSN (C++), various software built-ins. Go: custom/wrap.
   </td>
  </tr>
  <tr>
   <td><strong>Wavelet Noise</strong>
   </td>
   <td>Subtracting up/down-sampled random image from original <sup>35</sup>
   </td>
   <td>Nearly perfect band-limiting, reduced aliasing <sup>35</sup>
   </td>
   <td>Layered terrain strata, distinct cloud layers, stratified resources <sup>8</sup>
   </td>
   <td>Pixar paper (C++). Go: build from wavelet transform libs.
   </td>
  </tr>
  <tr>
   <td><strong>Curl Noise</strong>
   </td>
   <td>Curl of a noise-based potential field <sup>10</sup>
   </td>
   <td>Divergence-free vector fields, turbulent/swirling patterns <sup>10</sup>
   </td>
   <td>Air/water circulation (wind, ocean currents), fluid dynamics for climate, particle advection in erosion <sup>12</sup>
   </td>
   <td>C++/GLSL libs exist. Go: custom on base noise.
   </td>
  </tr>
  <tr>
   <td><strong>Flow Noise</strong>
   </td>
   <td>Time-varying noise, often by rotating gradients and/or advecting domain <sup>59</sup>
   </td>
   <td>Dynamic, flowing, swirling animated patterns, can be tiling <sup>59</sup>
   </td>
   <td>Animated air/water currents, evolving weather/clouds, dynamic erosion visualization <sup>59</sup>
   </td>
   <td>GLSL implementations exist. Go: custom.
   </td>
  </tr>
  <tr>
   <td><strong>Anisotropic Noise</strong>
   </td>
   <td>Noise with directional statistical properties <sup>35</sup>
   </td>
   <td>Directional patterns (vs. isotropic) <sup>61</sup>
   </td>
   <td>Directional terrain (dunes, glacial scouring), oriented resource deposits, wood grain <sup>41</sup>
   </td>
   <td>Gabor libraries; coordinate scaling on isotropic noise.
   </td>
  </tr>
  <tr>
   <td><strong>Domain Warping</strong>
   </td>
   <td>Distorting input coordinates of a noise function using another noise <sup>64</sup>
   </td>
   <td>Complex, organic, turbulent, breaks regularity <sup>64</sup>
   </td>
   <td>Deformed terrain, complex river meanders, irregular resource veins/clusters, varied biome patterns <sup>64</sup>
   </td>
   <td>FastNoiseLite (Go port), noise-rs (Rust).
   </td>
  </tr>
  <tr>
   <td><strong>Multi-Fractal Noise</strong>
   </td>
   <td>fBm with varying parameters or more complex octave combination <sup>33</sup>
   </td>
   <td>Heterogeneous fractal patterns, spatially varying roughness/detail <sup>33</sup>
   </td>
   <td>Realistic varied terrain (smooth plains to jagged peaks), complex clouds, variable rock erodibility <sup>25</sup>
   </td>
   <td>FastNoiseLite (Go port), noise-rs (Rust).
   </td>
  </tr>
  <tr>
   <td><strong>Pattern-Based / Hybrid</strong>
   </td>
   <td>Combining deterministic patterns (e.g., L-Systems) with noise modulation <sup>79</sup>
   </td>
   <td>Structured yet varied outputs, combines grammar with organic feel <sup>1</sup>
   </td>
   <td>Branching ore veins/rivers (L-Systems + noise), noise-guided feature placement <sup>79</sup>
   </td>
   <td>Custom L-System + noise libs.
   </td>
  </tr>
  <tr>
   <td><strong>Spot Noise</strong> (Interpreted)
   </td>
   <td>Distribution of localized noise kernels/impulses <sup>35</sup>
   </td>
   <td>Localized feature contributions, density-controlled patterns
   </td>
   <td>Discrete resource nodes, small terrain details (boulders), localized weathering.
   </td>
   <td>Sparse convolution/Gabor libs.
   </td>
  </tr>
</table>



## 

---
4. Application of Noise Algorithms to Specific World Generation Systems

The true power of a diverse noise toolkit emerges when specific algorithms are strategically applied to model the distinct characteristics of various interconnected world systems.


### 4.1. Tectonic Plates & Continental Drift

The generation of believable tectonic plates and the simulation of their interactions to form large-scale geological features like continents, mountain ranges, and rifts is a complex challenge. Noise algorithms play a critical role at multiple stages.

Defining Plate Shapes and Boundaries:

The initial definition of tectonic plate shapes often utilizes algorithms that can partition space into distinct, irregular regions.



* **Worley Noise (Cellular Noise):** This is a common starting point for delineating initial plate regions.<sup>13</sup> The F1​ distance (distance to the nearest seed point) can define the core extent of individual plates. The boundaries between these Voronoi cells naturally form the initial fault lines. The F2​−F1​ variation is particularly adept at highlighting these boundaries directly, offering a field that can be used to define the width or characteristics of these initial fault zones. One described approach involves using Voronoi cells with Simplex noise modulating the distance function to achieve more organic plate outlines.<sup>28</sup>
* **Perlin/Simplex Noise (fBm):** Low-frequency Perlin or Simplex noise, when thresholded, can also generate large, contiguous "plate-like" regions.<sup>13</sup> More commonly, these noises are used in conjunction with Worley noise to add irregularity and a more natural, less geometric appearance to the Voronoi-defined plate boundaries.<sup>14</sup>
* **Domain Warping:** Applying domain warping to the coordinate system before sampling Worley or Perlin/Simplex noise can introduce large-scale, non-linear distortions to plate shapes, simulating a history of deformation or complex interactions that are not easily captured by simpler noise combinations.

Simulating Plate Interactions (Rifts, Mountains, Subduction Zones):

Once initial plate regions are defined, their interactions are driven by simulated motion, which can also be noise-driven.



* **Motion Vectors (Vector Value Noise / Curl Noise / Flow Noise):** Each plate can be assigned a motion vector (direction and speed). This vector can be derived from sampling a 2D or 3D vector noise field. A custom "vector value noise" has been used for this purpose.<sup>14</sup> Alternatively, Curl Noise can generate divergence-free vector fields that represent mantle convection cells or large-scale flow patterns, which in turn drive plate motion.<sup>12</sup> Flow Noise could introduce time-varying components to these motion vectors. The relative motion of adjacent plates at their shared boundary (convergent, divergent, or transform) determines the type of geological features that form.
* **Feature Generation at Boundaries (e.g., "Mountain Envelope" Function):** The concept of a "mountain envelope" function, though not fully detailed in the provided material, suggests a procedural approach where the type and magnitude of terrain deformation (uplift for mountains, depression for trenches) at plate boundaries are determined by factors like the distance from the border, the shape of the border, and the relative velocities of the interacting plates.<sup>13</sup> Different noise profiles can be applied based on the type of fault:
    * **Convergent Boundaries (Mountains/Subduction):** Where plates collide, significant uplift occurs. Ridged Multifractal noise is excellent for sculpting sharp, young mountain ranges.<sup>25</sup> Smoother fBm or Perlin/Simplex noise can represent older, more eroded mountain belts. For subduction zones, where one plate dives beneath another, this function could create deep oceanic trenches bordered by volcanic arcs or coastal mountain ranges.<sup>7</sup>
    * **Divergent Boundaries (Rifts):** Where plates pull apart, rift valleys on land or mid-ocean ridges at sea are formed.<sup>7</sup> Noise can be used to define the width and depth of the rift, the jaggedness of the rift shoulders, and the associated volcanic activity.
    * **Transform Boundaries:** Characterized by lateral shearing. Noise can define the complexity of the fault zone and associated smaller-scale deformations.
* **Fractal Detailing (fBm, Ridged/Billow):** Regardless of the primary mechanism for uplift or depression, various fractal noise types are essential for adding detailed topography to these large-scale features.<sup>7</sup>

The interplay between Voronoi-defined plate regions and noise-driven motion vectors allows for emergent boundary types. Instead of explicitly programming "convergent boundary here, divergent there," these interactions arise naturally from the relative motions of the plates, leading to more diverse and often surprising tectonic landscapes. Furthermore, the geological "age" or erosion state of mountain ranges can be simulated by modulating noise parameters; younger, active ranges might use higher frequency and amplitude noise (like Ridged Multifractal), while older, heavily eroded ranges would employ smoother, lower-frequency noise. This modulation could even be tied to a global simulation time or a per-plate "activity" parameter.

Simulating Continental Drift:

True continental drift simulation is computationally intensive. However, aspects can be driven by noise. Long-term plate movements can be influenced by evolving 2D or 3D noise fields (e.g., time-varying Simplex or Curl noise) representing patterns of mantle convection.28 One project describes a "convection" force derived from the gradient of evolving Simplex noise, which contributes to plate spin and overall drift.28


### 4.2. Elevation and Terrain Sculpting

The generation of terrain elevation is a primary application of noise functions, building upon the large-scale forms established by tectonic processes.

**Base Elevation:**



* **Perlin/Simplex fBm:** This remains the standard for generating initial heightmaps.<sup>3</sup> Adjusting the number of octaves, lacunarity, and persistence allows control over the scale (from broad continental swells to localized hills) and roughness of the terrain.

Mountain Ranges:

As discussed in the tectonics section, mountain ranges often form at plate boundaries.



* **Ridged Multifractal Noise:** Particularly effective for creating the sharp, craggy peaks characteristic of young, tectonically active mountain ranges.<sup>25</sup>
* **Billow Noise:** Can be used for more rounded, rolling mountain forms or uplifted plateau edges.<sup>75</sup>
* **Domain Warping:** Essential for adding geological realism by introducing folds, faults, and non-uniformity to mountain chains, simulating the complex stresses involved in orogeny.<sup>64</sup>

Valleys, Canyons, and Gullies:

These features are often the result of erosional processes (see Section 4.4). However, noise can also be used to define their initial forms:



* **Noise Shaping Functions:** Applying mathematical functions to noise output can create canyon-like depressions. For example, pow(noise_value, power) can carve deeper features by amplifying certain parts of the noise range.<sup>96</sup>
* **Inverted Noise:** Inverted Billow noise can create networks of valleys.
* **Worley Noise (F2​−F1​):** The boundaries defined by F2​−F1​ Worley noise can serve as initial paths for river valleys or canyons before erosion further sculpts them.

Stratification and Layering:

Representing distinct geological layers is crucial for realistic erosion and resource distribution.



* **Wavelet Noise:** The band-limiting property of Wavelet noise makes it highly suitable for defining distinct horizontal strata.<sup>47</sup> Each frequency band can correspond to a different rock layer with unique properties (e.g., color, hardness, erodibility).
* **3D Noise Fields:** A 3D Perlin, Simplex, or Value noise field can be thresholded at different levels to create volumetric layers of various rock types.<sup>6</sup> This allows for complex, non-horizontal strata. The Far Cry 5 GDC presentation mentioned slicing geometry into strata chunks for cliffs, a related concept.<sup>102</sup>
* **Domain Warping on Strata:** Applying domain warping to the 3D noise field used for stratification can simulate folding and faulting of these layers, creating complex geological cross-sections.

A multi-scale strategy is vital for terrain. Low-frequency base noise (e.g., Simplex fBm) defines continental masses and large basins. Mid-frequency noise, often informed by tectonic outputs (e.g., Ridged Multifractal along convergent boundaries), sculpts major mountain ranges. Higher-frequency detail noise then adds general surface roughness and smaller features. The characteristics of the elevation model are paramount, as they directly influence subsequent climate simulations (e.g., temperature via altitude, rainfall via orographic effects <sup>30</sup>) and hydrological processes (defining watershed boundaries and river flow paths <sup>105</sup>). Unrealistic elevation patterns will inevitably lead to unrealistic climate and water systems.


### 4.3. Climate Modeling

Noise functions are instrumental in generating the spatial variability inherent in climatic factors.

**Energy Balance and Temperature Gradients:**



* **Base Temperature:** Latitude is the primary determinant of base temperature due to insolation angles.
* **Noise Modulation for Temperature:** This latitudinal gradient can be modulated by:
    * Elevation: Temperature decreases with altitude. A noise map representing elevation (from Section 4.2) is used to adjust local temperatures.<sup>16</sup>
    * Proximity to Water: Large water bodies moderate temperature; a distance field from oceans/large lakes can be used as a mask or modulator.
    * General Variation: Low-frequency Perlin or Simplex noise can introduce broad regional temperature variations (e.g., warmer or cooler continental interiors).
* **Solar Radiation Variability:** Cloud cover, itself procedurally generated using noise (see below), directly impacts the amount of solar radiation reaching the surface. Noise can introduce random fluctuations in cloud density or type, thereby creating spatial and temporal variability in insolation.<sup>109</sup>

**Air and Water Circulation Patterns:**



* **Curl Noise:** This is highly effective for generating 2D (surface winds, surface currents) or 3D (atmospheric circulation, ocean currents) vector fields that are divergence-free, mimicking incompressible fluid flow.<sup>10</sup> Layering octaves of Curl noise can produce multi-scale turbulence, from large gyres to smaller eddies.
* **Flow Noise:** For dynamic, animated wind and water currents, Flow noise offers time-varying patterns that can simulate evolving systems.<sup>59</sup>
* **Large-Scale Atmospheric Cells (Hadley, Ferrel, Polar):** While the fundamental structure of these cells is driven by global energy balance and Earth's rotation (Coriolis effect), their boundaries are not static or perfectly defined. Low-frequency 2D noise can be used to:
    * Define the average positions and strengths of persistent high and low-pressure zones that anchor these cells.<sup>67</sup>
    * Introduce irregularities and perturbations to the idealized cell boundaries and jet stream paths.
* **Cloud Patterns:**
    * **fBm (Perlin/Simplex):** Commonly used for generating static cloud cover maps, with density varying based on noise values.<sup>3</sup>
    * **3D/4D Noise:** For volumetric clouds that change over time, 3D noise (for density) sampled along a 4th (time) dimension can create evolving cloudscapes.<sup>3</sup>
    * **Domain Warping/Curl Noise:** Can be applied to cloud density fields to create more dynamic and turbulent cloud formations, such as swirling storm systems.<sup>66</sup>
    * **Procedural Cloudscape Models:** More advanced models combine implicit functions for cloud shapes with procedural noise for density and detail, allowing for different cloud types (cumulus, stratus, cirrus) based on altitude and atmospheric conditions.<sup>114</sup>

Rainfall and Humidity Distribution:

Rainfall is influenced by moisture sources (oceans), wind patterns carrying moisture, orographic lift over mountains, and temperature (affecting saturation).



* **Noise-Driven Factors:** Noise-generated wind patterns (Curl noise) transport moisture. Noise-generated elevation (fBm, Ridged) creates orographic effects.
* **Direct Noise Modulation:** Perlin or Simplex fBm can add regional variability to rainfall and humidity, simulating localized weather events or microclimates not captured by broader rules.<sup>30</sup> For instance, a humidity map can be generated using noise, then modified by proximity to water and areas of upwelling air.

Biome Placement and Transition Zones:

Biomes are typically determined by temperature and precipitation/humidity.



* **Climate-Based Placement:** Using the noise-generated temperature and rainfall maps, biomes can be assigned based on a lookup (similar to a Whittaker diagram).
* **Noise for Direct Biome Suitability:** Alternatively, separate noise fields can represent the "suitability" for each biome type. The biome with the highest suitability value at a given location is chosen.<sup>30</sup> Worley noise can be effective for creating distinct, large biome regions.<sup>29</sup>
* **Transition Zones (Ecotones):** To avoid sharp, unrealistic biome boundaries, noise can be used to blend biome characteristics. This can be achieved by:
    * Blending the noise values that determine biome types near boundaries.
    * Using a separate noise field to control the "mix" of vegetation or terrain features from adjacent biomes in transition zones.<sup>117</sup>

A powerful aspect of this system is the potential for emergent climatic phenomena. For example, noise-driven mountain ranges can realistically influence noise-driven wind patterns, leading to orographic precipitation that defines humid and arid regions, which in turn dictates biome distribution. This chain of interactions, all seeded by various noise functions, can produce complex and self-consistent climates. Furthermore, employing time-varying noise (e.g., 4D Simplex or Flow Noise) for elements like pressure cells or cloud cover can introduce dynamic weather systems, where patterns shift and evolve, impacting local conditions over simulated time.


### 4.4. Erosion and Watersheds

Erosion sculpts terrain based on material resistance and the action of erosive forces like water and wind. Noise can influence both aspects.

**Modulating Erosion Rates and Material Hardness:**



* **Rock Hardness Maps:** A 3D noise field (Perlin, Simplex, Worley for distinct inclusions, or Wavelet for stratified rock <sup>6</sup>) can represent the spatial variation of rock hardness or erodibility. This 3D map is sampled by the erosion simulation at the current terrain surface. Softer areas (lower noise values, perhaps) erode faster than harder areas.<sup>34</sup>
    * This approach allows for differential erosion, where more resistant rock formations (e.g., defined by higher values in the hardness noise map) remain as ridges or mesas, while softer materials are preferentially removed.
* **Spatially Varying Erosion Parameters:** Noise can directly modulate parameters of the erosion algorithm itself. For example, a 2D noise map could control the intensity of hydraulic erosion (e.g., "stream power") or thermal weathering in different regions, independent of just rock hardness. <sup>127</sup> mentions bias masks for rock softness and erosion strength.

Watershed Delineation and River Network Formation:

Watersheds and the resulting river networks are primarily determined by the topography of the elevation map. Standard hydrological algorithms (e.g., D8 flow accumulation) operate on this Digital Elevation Model (DEM).



* **Influence of DEM Noise Quality:** The characteristics of the noise used to generate the DEM (see Section 4.2) are crucial. Artifacts, unrealistic smoothness, or excessive ruggedness in the elevation noise will directly translate into unnatural watershed boundaries and river patterns.<sup>106</sup>
* **Perturbing Flow Paths:** Subtle, high-frequency noise can be added to the DEM before watershed analysis. This can introduce minor variations in flow paths, preventing unnaturally straight river segments without significantly altering the major topography.
* **Guiding River Networks:**
    * **L-Systems or Space Colonization Algorithms:** These can generate branching patterns suitable for river networks.<sup>83</sup> The paths generated can then be "carved" into the noise-based heightmap.
    * **Noise Modulation of Path Algorithms:** Noise can influence parameters within these path-generation algorithms, such as branching probability, segment length, or sinuosity.<sup>8</sup> For example, a noise field could make rivers meander more in flatter areas (identified by the elevation noise).
    * **Worley Noise for Initial Paths:** The F2-F1 variation of Worley noise can create ridge-like networks that could serve as initial high-ground guiding water flow or, conversely, valley-like networks if inverted.

The use of 3D noise to define subsurface rock hardness creates a particularly interesting dynamic. As erosion algorithms (like hydraulic erosion) carve into the terrain, they expose these underlying variations. This can lead to the emergence of features like cliffs where resistant layers are undercut, mesas capped by hard rock, or cuestas with gentle dip slopes and steep scarps, all reflecting the 3D noise-defined geology. The statistical properties of the noise used for the initial terrain (e.g., its fractal dimension) also implicitly influence the density and intricacy of the resulting stream networks; a more rugged, detailed terrain will naturally support a more complex drainage system.


### 4.5. Resource Distribution

Noise functions are widely used to determine the placement, density, and shape of various resources, from mineral ores to flora.

**Ore Veins and Clustered Deposits:**



* **3D Noise Thresholding:** A common method involves generating a 3D noise field (Perlin, Simplex, or Worley for cellular deposits) and placing resources where the noise value exceeds a specific threshold.<sup>3</sup> Different noise instances or seeds can be used for different ore types, and thresholds can be adjusted to control rarity. <sup>131</sup> suggests using Perlin noise for gold/diamonds by thresholding an overlaid noise cloud. <sup>135</sup> mentions combining Perlin and Worley noise for 3D cellular structures.
* **Domain Warping on 3D Noise:** Applying domain warping to the 3D noise field used for ore placement can create more convoluted, less uniform, and more geologically plausible vein structures or deposit shapes.
* **"Walker" Algorithm:** This agent-based approach involves "walkers" that traverse the 3D space, "painting" ore as they move. Their behavior (movement direction, radius of influence, splitting, termination) can be randomized or modulated by noise functions.<sup>130</sup> For example, a walker might be more likely to turn or deposit ore if it enters a region with a certain 3D noise value.
* **L-Systems with Noise Modulation:** L-systems are well-suited for generating the branching patterns characteristic of some ore veins.<sup>79</sup> Noise can then be used to control parameters like vein thickness along its length, the probability of branching at any point, the angle of branches, or the concentration of ore within the vein.
* **3D Gabor Noise:** For ore bodies or veins that exhibit a strong directional preference or anisotropy (e.g., aligned with geological faults or strata), 3D Gabor noise can be used to define their shape and extent.<sup>37</sup>

Resource Density Variation:

To avoid uniform resource distribution, a large-scale, low-frequency noise map can define broad regions of higher or lower overall resource abundance. This density map can then act as a multiplier or modulator for the more detailed noise or algorithms used for specific deposit placement. For example, a region with a high value in the density map might have a lower threshold for ore generation or a higher probability of walker/L-system initiation.

Shaped Resource Clusters:

While noise functions generate the raw data, other algorithms can refine the placement into more coherent clusters. DBSCAN (Density-Based Spatial Clustering of Applications with Noise) is a clustering algorithm that can identify groups of points (potential resource locations) that are closely packed together, forming clusters of arbitrary shape, and can identify isolated points as "noise" (outliers).147 This could be used as a post-processing step: generate initial candidate resource locations using a noise threshold, then use DBSCAN to group these into more naturally shaped deposits and perhaps discard sparse outliers.

The geological context derived from tectonic and erosion simulations should ideally inform resource distribution. For instance, certain minerals are often associated with volcanic activity (defined by tectonics and elevation) or concentrated by hydrothermal processes along fault lines. Noise functions used for resource placement can be biased or masked by these geological features. Rather than applying resource generation noise globally, the outputs from the tectonic, elevation, and erosion systems can create probability fields or masks. For example, the likelihood of finding specific ores could be increased if a 3D noise sample falls within a region identified as a fault zone, an area with specific rock strata (defined by Wavelet or 3D noise), or near volcanic intrusions.

For vein-like resources, a hybrid approach combining pathfinding algorithms (like A* or random walks) or L-systems with 3D noise for defining thickness and local deviations can yield more convincing results than simple 3D thresholded noise. The path itself could be guided by a low-frequency 3D noise field indicating "favorability" for vein formation, perhaps related to stress fields or pre-existing fractures in the rock.

**Table 3: Noise Algorithm Suitability Matrix for World Systems**


<table>
  <tr>
   <td><strong>World System Element</strong>
   </td>
   <td><strong>Recommended Primary Noise(s)</strong>
   </td>
   <td><strong>Secondary/Modulating Noise(s)</strong>
   </td>
   <td><strong>Key Techniques/Parameters</strong>
   </td>
   <td><strong>Rationale/Justification</strong>
   </td>
  </tr>
  <tr>
   <td><strong>Tectonic Plates (Shapes/Boundaries)</strong>
   </td>
   <td>Worley (F1​) <sup>28</sup>, Low-Freq Perlin/Simplex <sup>13</sup>
   </td>
   <td>Perlin/Simplex (for distortion) <sup>14</sup>
   </td>
   <td>Seed points, distance metrics, thresholding, domain warping
   </td>
   <td>Cellular structure for plates, smooth noise for organic boundaries.
   </td>
  </tr>
  <tr>
   <td><strong>Tectonic Interactions (Mountains, Rifts)</strong>
   </td>
   <td>Ridged Multifractal <sup>75</sup>, Curl Noise (for motion) <sup>12</sup>, "Mountain Envelope" (conceptual) <sup>13</sup>
   </td>
   <td>fBm (for detail), Value Noise (for blockiness)
   </td>
   <td>Plate motion vectors, convergence/divergence logic, noise shaping
   </td>
   <td>Sharp peaks for young mountains, vector fields for plate movement.
   </td>
  </tr>
  <tr>
   <td><strong>Elevation (Base Terrain)</strong>
   </td>
   <td>Simplex fBm <sup>6</sup>, Perlin fBm <sup>33</sup>
   </td>
   <td>Domain Warping <sup>64</sup>
   </td>
   <td>Octaves, lacunarity, persistence
   </td>
   <td>Smooth, natural, multi-scale terrain. Warping for realism.
   </td>
  </tr>
  <tr>
   <td><strong>Elevation (Strata)</strong>
   </td>
   <td>Wavelet Noise <sup>47</sup>, 3D Value/Perlin Noise
   </td>
   <td>-
   </td>
   <td>Band-limited components, 3D thresholding
   </td>
   <td>Distinct geological layers, variable erodibility.
   </td>
  </tr>
  <tr>
   <td><strong>Climate (Temperature)</strong>
   </td>
   <td>Latitudinal Gradient + Perlin/Simplex fBm
   </td>
   <td>Noise from Elevation/Cloud Cover
   </td>
   <td>Altitude effect, proximity to water
   </td>
   <td>Realistic temperature variations based on geography and atmospheric conditions.
   </td>
  </tr>
  <tr>
   <td><strong>Climate (Air/Water Circulation)</strong>
   </td>
   <td>Curl Noise <sup>10</sup>, Flow Noise (animated) <sup>59</sup>
   </td>
   <td>Perlin/Simplex (for potential field or perturbation)
   </td>
   <td>Vector field generation, gradient rotation, advection
   </td>
   <td>Divergence-free, turbulent flow for winds, ocean currents, gyres.
   </td>
  </tr>
  <tr>
   <td><strong>Climate (Rainfall/Humidity)</strong>
   </td>
   <td>Perlin/Simplex fBm (modulated by circulation & orographics)
   </td>
   <td>Worley (for regional influence)
   </td>
   <td>Orographic lift logic, distance to moisture sources
   </td>
   <td>Spatially varied precipitation influenced by wind and terrain.
   </td>
  </tr>
  <tr>
   <td><strong>Climate (Biomes)</strong>
   </td>
   <td>Worley (for distinct regions) <sup>30</sup>, Perlin/Simplex (for temp/humidity maps)
   </td>
   <td>Noise for transition blending <sup>117</sup>
   </td>
   <td>Thresholding climate maps, suitability functions
   </td>
   <td>Placement based on climate conditions, smooth transitions.
   </td>
  </tr>
  <tr>
   <td><strong>Erosion (Material Hardness)</strong>
   </td>
   <td>3D Perlin/Simplex/Worley/Wavelet <sup>6</sup>
   </td>
   <td>-
   </td>
   <td>3D noise field sampled at surface
   </td>
   <td>Spatially varying rock resistance to erosion.
   </td>
  </tr>
  <tr>
   <td><strong>Erosion (Flow Guidance)</strong>
   </td>
   <td>Curl Noise (for water flow vectors)
   </td>
   <td>Noise on DEM (for path perturbation)
   </td>
   <td>Particle systems, hydraulic erosion algorithms
   </td>
   <td>Natural water flow paths influencing erosion patterns.
   </td>
  </tr>
  <tr>
   <td><strong>Watersheds</strong>
   </td>
   <td>(Derived from DEM)
   </td>
   <td>Noise on DEM (for minor path variation)
   </td>
   <td>Flow accumulation algorithms
   </td>
   <td>Defines drainage basins based on noise-generated topography.
   </td>
  </tr>
  <tr>
   <td><strong>Resources (Veins/Clusters)</strong>
   </td>
   <td>3D Perlin/Simplex/Worley <sup>130</sup>, L-Systems <sup>79</sup>, Gabor Noise (anisotropic)
   </td>
   <td>Noise for L-System parameters, Domain Warping
   </td>
   <td>Thresholding, walker algorithms, branching rules, anisotropy
   </td>
   <td>Varied ore body shapes, from clusters to branching veins.
   </td>
  </tr>
  <tr>
   <td><strong>Resources (Density)</strong>
   </td>
   <td>Low-frequency Perlin/Simplex
   </td>
   <td>-
   </td>
   <td>Multiplicative blending with placement noise
   </td>
   <td>Regional variations in resource abundance.
   </td>
  </tr>
</table>



## 

---
5. Survey of Code Libraries for Noise Generation

The choice of programming language and available libraries significantly impacts the ease of implementation and performance of procedural noise generation.


### 5.1. Go Libraries

For projects developed in Go, several native libraries offer foundational noise capabilities:



* **github.com/cjslep/noise**: This library provides 2D implementations of Perlin and Simplex noise. It also includes an octave-based noise feature (similar to fBm with constant gain and lacunarity) and an experimental Perlin noise variant using Catmull-Rom spline interpolation, though the latter is noted to have visual artifacts.<sup>151</sup> This library is suitable for basic 2D Perlin and Simplex noise requirements but lacks support for more advanced types like Worley, Gabor, or Curl noise.
* **github.com/larspensjo/Go-simplex-noise**: This library offers 1D, 2D, 3D, and 4D Simplex noise, based on Ken Perlin's reference implementations.<sup>154</sup> Its multi-dimensional Simplex support is valuable for various aspects of world generation, including volumetric textures or time-varying noise.
* **github.com/KEINOS/go-noise**: This package serves as a wrapper for go-perlin and opensimplex-go, providing access to Perlin and OpenSimplex noise in up to 3 dimensions.<sup>146</sup>
* **Wavelet Transform Libraries for Go**: Libraries such as github.com/octu0/wavelet (providing Haar wavelets) <sup>155</sup> and github.com/goccmack/godsp (offering Discrete Wavelet Transform, specifically Daubechies 4) <sup>156</sup> exist. These libraries provide the core wavelet transforms. To implement Wavelet Noise as described by Cook and DeRose <sup>47</sup>, which involves specific image processing steps (downsampling, upsampling, subtraction of noise tiles), one would need to build upon these transform primitives.
* **Libraries for Advanced Noise Types (Worley, Gabor, Curl, Flow) in Go**: Dedicated, feature-rich Go libraries for these more specialized noise types were not prominently identified in the surveyed materials.<sup>24</sup> Implementing these would likely require custom Go code based on algorithmic descriptions or by wrapping existing C/C++ libraries using CGo.


### 5.2. Portable Libraries (C++, Rust, etc.) with Potential Go Integration

Several highly portable noise libraries written in languages like C++ or Rust offer extensive features and sometimes have Go ports or are designed for easy wrapping.



* **FastNoiseLite**: This is an extremely portable open-source noise generation library designed for high performance and ease of integration across many languages, including an official Go port.<sup>169</sup>
    * **Supported Noise Types:** OpenSimplex2, OpenSimplex2S, Cellular (Worley), Perlin, Value, and ValueCubic.<sup>169</sup>
    * **Cellular (Worley) Features:** Offers various distance functions (Euclidean, EuclideanSquared, Manhattan, Hybrid) and crucially provides multiple return types including Distance (F1), Distance2 (F2), and combinations like Distance2Add (F1+F2) and Distance2Sub (F2-F1), allowing direct access to these useful values.<sup>26</sup>
    * **Fractal Types:** Supports FBM, Ridged, PingPong, and DomainWarpProgressive/Independent fractal types.<sup>26</sup>
    * **Domain Warp:** Includes OpenSimplex2-based and Basic Grid Gradient domain warp capabilities.<sup>169</sup>
    * **Relevance:** Highly significant for this project due to its Go port, comprehensive feature set covering many requested noise types (including detailed Worley outputs and various fractal/domain warp options), and focus on performance and portability.
* **noise-rs (Rust)**: A comprehensive procedural noise generation library for Rust.<sup>24</sup>
    * **Supported Noise Types:** Includes Perlin, OpenSimplex, Simplex, SuperSimplex, Value, and Worley noise.
    * **Fractal/Modifiers:** Offers an extensive suite of combiners and modifiers such as Abs, Add, Blend, Displace (for domain warping), Exponent, FBM, Billow, HybridMulti, RidgedMulti, Multiply, Power, RotatePoint, ScalePoint, Select, Terrace, and Turbulence (also for domain warping).
    * **Relevance:** While a Rust library, its rich feature set for combining and modifying noise makes it a valuable reference. Specific modules could potentially be ported to Go if needed, or it could be used if the project incorporates Rust components.
* **libnoise (C++)**: A well-established C++ library for generating coherent noise, including Perlin noise and ridged multifractal noise, with a modular system for combining noise sources.<sup>2</sup>
    * **Relevance:** Useful if C++ interoperation is feasible, though it may lack some of the newer noise types found in libraries like FastNoiseLite or noise-rs.
* **OpenSN (C++)**: Mentioned as an open-source synthetic noise library that includes various noise algorithms, including a novel "prime gradient noise," alongside analysis tools.<sup>5</sup>
    * **Relevance:** Could be a source for exploring less common or more advanced algorithms if direct C++ usage or wrapping is an option.


### 5.3. GPU-Accelerated Options (GLSL/HLSL)

For performance-critical aspects, implementing noise generation directly on the GPU using shader languages can be beneficial.



* Many noise algorithms, including Perlin, Simplex, Worley, and Flow noise, have been implemented in GLSL or HLSL for real-time applications.<sup>11</sup>
* Unreal Engine, for example, provides material-based procedural noise, including Voronoi, and offers a "Fast Gradient - 3D Texture" option for optimized performance.<sup>181</sup>
* The JCGT paper by Gustavson et al. on Tiling Simplex Noise and Flow Noise provides direct GLSL implementations.<sup>59</sup>
* **Relevance:** GPU implementations are most relevant for tasks that can be parallelized and benefit from rapid evaluation, such as real-time texture generation, visualization of fluid simulations, or dynamic terrain modification effects. For a Go-based project, this might involve interacting with a separate rendering engine or using compute shaders if the Go environment supports such interoperation.

The landscape of Go-native noise libraries provides good support for foundational algorithms like Perlin and Simplex. However, for more advanced or specialized types such as Worley noise with detailed Fn​ access, Gabor noise, Curl noise, or the specific Pixar-style Wavelet noise, developers may need to turn to more comprehensive, portable libraries like FastNoiseLite (which thankfully has a Go port) or consider custom implementations, potentially by wrapping C/C++ libraries via CGo. The "explicit" generation process of Wavelet noise, often involving pre-calculated tiles <sup>35</sup>, also presents an architectural consideration for infinite worlds, possibly requiring tiling strategies or dynamic local generation, contrasting with the on-the-fly evaluation typical of most other procedural noise functions.

**Table 4: Code Library Survey for Procedural Noise**


<table>
  <tr>
   <td><strong>Library Name</strong>
   </td>
   <td><strong>Primary Language(s)</strong>
   </td>
   <td><strong>Go Port</strong>
   </td>
   <td><strong>Key Noise Types Supported</strong>
   </td>
   <td><strong>Link/Source</strong>
   </td>
   <td><strong>License (Commonly)</strong>
   </td>
   <td><strong>Portability Notes</strong>
   </td>
  </tr>
  <tr>
   <td>cjslep/noise
   </td>
   <td>Go
   </td>
   <td>Native
   </td>
   <td>Perlin (2D), Simplex (2D), Octave (fBm-like)
   </td>
   <td><sup>151</sup>
   </td>
   <td>GPL-3.0
   </td>
   <td>Go only
   </td>
  </tr>
  <tr>
   <td>larspensjo/Go-simplex-noise
   </td>
   <td>Go
   </td>
   <td>Native
   </td>
   <td>Simplex (1D-4D)
   </td>
   <td><sup>154</sup>
   </td>
   <td>Unspecified (likely open source)
   </td>
   <td>Go only
   </td>
  </tr>
  <tr>
   <td>KEINOS/go-noise
   </td>
   <td>Go
   </td>
   <td>Native
   </td>
   <td>Perlin (up to 3D), OpenSimplex (up to 3D) (wrappers)
   </td>
   <td><sup>146</sup>
   </td>
   <td>MIT
   </td>
   <td>Go only
   </td>
  </tr>
  <tr>
   <td><strong>FastNoiseLite</strong>
   </td>
   <td>C#, C++, Java, JS, Rust, <strong>Go</strong>, HLSL, GLSL, etc.
   </td>
   <td>Yes (Official)
   </td>
   <td>OpenSimplex2, OpenSimplex2S, Cellular (Worley with F1​,F2​,F2​−F1​, etc.), Perlin, Value, ValueCubic. Fractals: FBM, Ridged, PingPong. Domain Warp.
   </td>
   <td><sup>169</sup>
   </td>
   <td>MIT
   </td>
   <td>Highly portable
   </td>
  </tr>
  <tr>
   <td>noise-rs
   </td>
   <td>Rust
   </td>
   <td>No (but Rust can interop with Go via C ABI)
   </td>
   <td>Perlin, OpenSimplex, Simplex, SuperSimplex, Value, Worley. Extensive modifiers: FBM, Billow, RidgedMulti, HybridMulti, Displace/Turbulence (Domain Warp).
   </td>
   <td><sup>75</sup>
   </td>
   <td>Apache-2.0 / MIT
   </td>
   <td>Rust, potential for C interop
   </td>
  </tr>
  <tr>
   <td>libnoise
   </td>
   <td>C++
   </td>
   <td>No (would require CGo)
   </td>
   <td>Perlin, Ridged Multifractal, various combiners.
   </td>
   <td><sup>176</sup>
   </td>
   <td>LGPL
   </td>
   <td>C++
   </td>
  </tr>
  <tr>
   <td>OpenSN
   </td>
   <td>C++
   </td>
   <td>No (would require CGo)
   </td>
   <td>Prime Gradient Noise, other noise types, analysis tools.
   </td>
   <td><sup>72</sup>
   </td>
   <td>Unspecified (research)
   </td>
   <td>C++
   </td>
  </tr>
  <tr>
   <td>Pixar Wavelet Noise
   </td>
   <td>C++ (example)
   </td>
   <td>No (would require porting/CGo)
   </td>
   <td>Wavelet Noise (specific band-limited method).
   </td>
   <td><sup>47</sup>
   </td>
   <td>(Pixar internal, concept public)
   </td>
   <td>C++ algorithm description
   </td>
  </tr>
  <tr>
   <td>Bridson Curl Noise
   </td>
   <td>(Algorithm)
   </td>
   <td>No (would require implementation)
   </td>
   <td>Curl Noise (from potential field).
   </td>
   <td><sup>10</sup>
   </td>
   <td>(Algorithm description)
   </td>
   <td>Algorithm
   </td>
  </tr>
  <tr>
   <td>JCGT Flow Noise
   </td>
   <td>GLSL
   </td>
   <td>No (shader code)
   </td>
   <td>Tiling Simplex Noise, Flow Noise (2D, 3D) with analytical gradients.
   </td>
   <td><sup>59</sup>
   </td>
   <td>MIT (likely for code)
   </td>
   <td>GLSL shaders
   </td>
  </tr>
</table>



## 

---
6. Specific Algorithm Deep Dives

This section provides a more granular examination of selected advanced noise algorithms, focusing on their generation process and key characteristics pertinent to the project.


### 6.1. Gabor Noise: Algorithmic Breakdown

Gabor noise belongs to the category of sparse convolution noises.<sup>35</sup> Its distinctive anisotropic patterns arise from convolving sparsely distributed random impulses with Gabor kernels.



* **Sparse Impulses:** The process begins by distributing a set of points (impulses) randomly, but sparsely, in the domain (e.g., 2D or 3D space). Each impulse typically has a random weight and potentially other properties.<sup>39</sup>
* **Gabor Kernel:** The Gabor kernel is a product of a Gaussian function and a sinusoidal function.<sup>37</sup>
    * The Gaussian component provides localization, ensuring the kernel's influence is concentrated around its center. Its parameters control the size and falloff (e.g., sigma or a for width/bandwidth <sup>37</sup>).
    * The sinusoidal component introduces an oriented wave pattern. Its parameters control the frequency (F0​) and orientation (ω0​) of the wave.<sup>37</sup>
    * The combination results in a localized, oriented ripple or "spot."
* **Convolution Summation:** To evaluate the Gabor noise at a sample point P, the contributions of all Gabor kernels whose centers are within a certain radius of P are summed. The contribution of each kernel is its value at the displacement vector (P−kernel_center), weighted by the impulse's random weight.<sup>44</sup> GaborNoise(P)=∑i​wi​⋅KernelGabor​(P−Ci​,F0,i​,ω0,i​,ai​,Ki​) where Ci​ is the center of the i-th impulse, wi​ its weight, and F0,i​,ω0,i​,ai​,Ki​ are its Gabor kernel parameters.
* **Anisotropy and Spectral Control:** The key strength of Gabor noise is its direct control over anisotropy (via the orientation ω0​ of the sinusoid) and frequency content (via F0​ and bandwidth a).<sup>37</sup> By using multiple sets of impulses with different Gabor kernel parameters (e.g., different orientations or frequencies), complex, multi-layered anisotropic textures can be built.<sup>142</sup> The "sparse" nature implies that for any given sample point, only a limited number of nearby impulses contribute significantly to the final noise value. This locality is a key difference from lattice noises like Perlin or Simplex, where evaluation always involves the surrounding lattice cell. The performance of Gabor noise can thus depend on the density of impulses and the support radius of the Gabor kernel; efficient spatial data structures are often needed to quickly find contributing impulses for large-scale generation.<sup>182</sup>


### 6.2. Wavelet Noise (Pixar Method): Algorithmic Breakdown

The Wavelet Noise method described by Cook and DeRose (often referred to as the Pixar method) aims to create noise that is well band-limited, reducing aliasing and improving control over detail across different scales.47

The core idea is to generate noise bands, where each band contains energy only within a specific range of frequencies.



* **Generation of a Single Noise Band (N):**
    1. **Create Random Tile (R):** Start with a tile (e.g., a 2D image or 3D volume) filled with random, uncorrelated values (white noise).<sup>47</sup>
    2. **Downsample (R↓):** Downsample the random tile R by a factor of 2 in each dimension to create a half-size tile R↓. This step typically involves a low-pass filter to prevent aliasing during downsampling.
    3. **Upsample (R↓↑):** Upsample the tile R↓ back to the original dimensions of R, creating R↓↑. This step uses an interpolation filter.
    4. **Subtract:** The noise band N is then computed as the difference: N=R−R↓↑.<sup>47</sup> This subtraction removes the lower-frequency components present in R↓↑, leaving N with primarily higher-frequency details that were lost in the downsampling/upsampling process. This N is one band of noise.
* **Tiling and Evaluation:** The generated noise band tile N is typically designed to be tilable. To evaluate the noise at an arbitrary point (x,y,z), the corresponding point within the (potentially repeating) tile N is sampled, often using interpolation (e.g., trilinear) between the discrete tile values.
* **Multiresolution Summation (M(x)):** Similar to fBm, the final Wavelet noise M(x) is constructed by summing multiple scaled and attenuated versions of these band-limited noise tiles Nb​(x). Each Nb​(x) corresponds to a different frequency band (octave), achieved by further scaling the input coordinates or by generating different N tiles from progressively downsampled versions of the initial random tile R. M(x)=∑b=bmin​bmax​​wb​Nb​(2bx) where wb​ are weights that control the amplitude of each frequency band, allowing for precise spectral shaping.<sup>47</sup> The pre-computation of noise tiles makes Wavelet noise an "explicit" noise. For infinitely large worlds, strategies for seamlessly tiling these precomputed bands or adapting the generation to be more on-the-fly for local regions are crucial architectural considerations.


### 6.3. Curl Noise: Algorithmic Breakdown

Curl noise generates a divergence-free vector field, making it ideal for simulating incompressible fluid flow, by taking the curl of a potential field ψ.<sup>10</sup>



* **Potential Field (ψ):**
    * In 2D, ψ is a scalar field, often generated by a 2D noise function like Perlin or Simplex noise: ψ(x,y)=Noise(x,y).
    * In 3D, ψ is a vector field ψ​=(ψx​,ψy​,ψz​). Each component can be derived from a 3D noise function, often by sampling the same noise function with large offsets for each component to ensure they are reasonably uncorrelated: ψx​(x,y,z)=Noise(x,y,z) ψy​(x,y,z)=Noise(x+offsetx​,y+offsety​,z+offsetz​) ψz​(x,y,z)=Noise(x+offsetx′​,y+offsety′​,z+offsetz′​) Typically, multiple octaves (fBm) are used for each component of ψ​ to create turbulent potential fields.
* **Curl Calculation (v=∇×ψ​):**
    * **In 2D:** The velocity field v=(vx​,vy​) is given by: vx​=∂y∂ψ​ vy​=−∂x∂ψ​
    * **In 3D:** The velocity field v=(vx​,vy​,vz​) is given by: vx​=∂y∂ψz​​−∂z∂ψy​​ vy​=∂z∂ψx​​−∂x∂ψz​​ vz​=∂x∂ψy​​−∂y∂ψx​​
* **Derivative Approximation (Finite Differences):** If the base noise library does not provide analytical derivatives of the potential field ψ, the partial derivatives must be approximated using finite differences. For a derivative like ∂y∂ψz​​, a central difference approximation is common: ∂y∂ψz​​≈2ϵψz​(x,y+ϵ,z)−ψz​(x,y−ϵ,z)​ where ϵ is a small step size.<sup>10</sup> Using analytical derivatives, if available from the noise library, is generally preferred for accuracy and potentially performance.<sup>59</sup>
* **Boundary Handling:** To make the flow respect solid boundaries (e.g., terrain), the potential field ψ can be modulated near boundaries. For an inviscid boundary condition (v⋅n=0), the potential can be ramped so that the boundary becomes an isocontour of ψ (in 2D) or by specific manipulations of tangential and normal components of ψ​ (in 3D).<sup>10</sup>

The quality of Curl noise heavily depends on the quality and differentiability of the underlying potential noise field. Using base noise that provides analytical gradients can significantly improve the accuracy and smoothness of the resulting flow fields compared to relying on finite difference approximations.


### 6.4. Flow Noise (Tiling Simplex Variant): Algorithmic Breakdown

Flow noise, particularly the tiling Simplex variant described by Gustavson et al., creates animated swirling patterns by modifying the Simplex noise generation process over time.<sup>59</sup>



* **Tiling Simplex Noise Base:** The algorithm starts with a Simplex noise implementation that is modified to tile seamlessly over specified integer periods. This involves adjustments to how grid coordinates are handled and how gradients are selected or wrapped at the tile boundaries.<sup>59</sup>
* **Time-Varying Gradient Rotation:** The core of the "flow" effect comes from dynamically rotating the pseudo-random gradient vectors associated with the Simplex grid vertices.
    * **In 2D:** The gradients (which are 2D vectors) are rotated in the plane by an angle α that varies with time (e.g., α(t)=speed⋅t). If the base gradients are generated from a pseudo-random angle (e.g., (cosθ,sinθ)), this rotation can be achieved by simply adding α(t) to θ before computing the sine and cosine.<sup>60</sup>
    * **In 3D:** The 3D gradient vectors are rotated around a pseudo-random axis (itself potentially derived from a hash of the vertex coordinates) by a time-varying angle α(t).<sup>59</sup>
* **Domain Advection (Optional Enhancement):** To further enhance the swirling or flowing appearance, the input coordinates (x,y,z) can be advected (displaced) slightly over time. This advection is often performed along the direction of the (time-varying) noise gradient itself, or perpendicular to it, creating a feedback loop where the noise pattern influences its own evolution. The JCGT paper mentions this as a texture domain warp along the gradient of the noise field.<sup>59</sup>
* **Analytical Derivatives:** This variant of Flow noise can also be formulated to provide exact analytical gradients with respect to the spatial coordinates, which is useful for bump mapping, lighting calculations, or deriving further vector fields (like Curl noise from Flow noise potential).<sup>59</sup>

This technique provides temporally coherent, animated noise that can tile, making it suitable for looping effects or dynamic textures on surfaces.


## 

---
7. Conclusions and Recommendations

This investigation into procedural noise algorithms reveals a rich and diverse landscape of techniques applicable to the multifaceted challenge of comprehensive world generation. The user's project, aiming to simulate interconnected systems from tectonic plates and climate to erosion and resource distribution, stands to benefit significantly from a carefully selected and integrated suite of noise functions, moving beyond a monolithic noise module.

**Key Findings:**



1. **No Single Noise Fits All:** The foundational algorithms (Perlin, Simplex, Value, Worley, fBm) each possess unique strengths and weaknesses. Perlin and Simplex noise are excellent for smooth, organic forms like base terrain and general scalar fields (temperature, rainfall), with Simplex generally offering superior quality and performance, especially in higher dimensions and for reducing grid artifacts.<sup>6</sup> Value noise provides simpler, blockier patterns suitable for specific effects or as a distinct base for fBm.<sup>14</sup> Worley noise excels at generating cellular structures, ideal for defining discrete regions like tectonic plates, biome cores, or specific resource deposit shapes (F1​,F2​,F2​−F1​ variations offering distinct structural primitives).<sup>9</sup> fBm remains essential for adding multi-scale detail to almost any noise-driven system.<sup>32</sup>
2. **Advanced Algorithms Offer Specialized Control:**
    * **Gabor Noise:** Its controllable anisotropy and frequency make it invaluable for directional features like wind-swept terrain, wood grain, or aligned mineral veins.<sup>37</sup>
    * **Wavelet Noise:** The band-limited nature offers precise spectral control, ideal for creating distinct geological strata or layered atmospheric effects with sharper transitions than standard fBm.<sup>35</sup>
    * **Curl Noise:** Its ability to generate divergence-free vector fields is fundamental for physically plausible simulations of air and water circulation (wind, ocean currents), forming a critical component of a dynamic climate model.<sup>10</sup>
    * **Flow Noise:** Provides temporally coherent, animated patterns suitable for dynamic visualization of fluid flows or evolving textures.<sup>59</sup>
    * **Domain Warping:** A powerful meta-technique to introduce complexity and organic distortions to any base noise, crucial for breaking up regularity and simulating deformation processes.<sup>64</sup>
    * **Multi-Fractal Noise:** Allows for spatially varying fractal dimensions, leading to more heterogeneous and realistic terrain where roughness and detail levels change across the landscape.<sup>33</sup>
    * **Pattern-Based/Hybrid Approaches (e.g., L-Systems + Noise):** Offer a way to combine rule-based structures with natural variation, excellent for features like branching river networks or ore veins.<sup>79</sup>
3. **System Interdependencies Drive Noise Selection:** The choice of noise for one system (e.g., elevation) has cascading effects on others (e.g., climate, watersheds, erosion). For instance, realistic orographic rainfall requires believable mountain ranges generated by appropriate elevation noise, which in turn are influenced by tectonic simulations that themselves can use noise for plate definition and movement. This underscores the need for a holistic approach to noise selection rather than optimizing for each system in isolation.
4. **Go Library Landscape:** While Go offers native libraries for foundational noises like Perlin and Simplex (e.g., cjslep/noise, larspensjo/Go-simplex-noise), support for more advanced types like Gabor, Curl, or the specific Pixar-style Wavelet noise is limited. **FastNoiseLite** stands out as a highly portable library with an official Go port, offering excellent support for Cellular (Worley) noise (including F1​,F2​,F2​−F1​ outputs), various fractal types (FBM, Ridged, PingPong), and domain warping.<sup>26</sup> For algorithms not covered by FastNoiseLite or other direct Go options, custom implementation or wrapping C/C++ libraries via CGo will likely be necessary.

**Recommendations:**



1. **Prioritize Simplex Noise over Perlin Noise:** For generating large-scale, smooth base features like continental landmasses, initial elevation maps, and broad climatic zones, Simplex noise should be favored due to its reduced directional artifacts and better performance in higher dimensions.<sup>13</sup>
2. **Leverage FastNoiseLite for Go Implementation:** Given its Go port and extensive feature set—including detailed Worley noise outputs (F1​,F2​,F2​−F1​), multiple fractal types (FBM, Ridged, PingPong), and domain warping capabilities—FastNoiseLite should be a primary candidate library for the project.<sup>26</sup> This will provide a robust foundation for many of the required noise patterns.
3. **Implement Curl Noise for Fluid Dynamics:** For air and water circulation, the project should implement Curl noise. This will likely require custom Go code that utilizes an underlying Simplex or Perlin noise library (like FastNoiseLite or larspensjo/Go-simplex-noise) to generate the potential field, and then computes the curl using finite differences or analytical derivatives if available from the base noise.<sup>10</sup> This is critical for realistic climate modeling.
4. **Explore Wavelet Noise for Stratification:** For generating distinct geological strata or layered cloud formations, investigate implementing Wavelet noise based on the Pixar method.<sup>47</sup> This may involve using Go's image processing capabilities or a Go wavelet transform library as a building block. The band-limited nature is key for sharp layer distinctions.
5. **Utilize Gabor Noise for Anisotropic Features:** For features requiring strong directionality (e.g., specific dune types, glacial scouring, fibrous resources, wood grain), consider implementing or wrapping a Gabor noise function. Its spectral control over orientation and frequency is superior to simple anisotropic scaling of isotropic noise.<sup>37</sup>
6. **Employ Domain Warping Extensively:** Use domain warping as a general technique to add complexity and naturalism to many features, from terrain deformation in tectonically active zones to irregular resource vein paths and varied biome patterns.<sup>64</sup> FastNoiseLite provides this capability.
7. **Consider Hybrid Approaches for Specific Structures:** For features like river networks or branching ore veins, explore combining L-systems (which would require a separate L-system library or implementation) with noise functions to modulate their growth parameters (thickness, branching angle, density).<sup>79</sup>
8. **Develop a Modular Noise System:** Given the variety of noise types and their specific applications, architect the project's noise module to be highly modular. This will allow for easier experimentation, substitution of different noise algorithms for specific tasks, and layering of noise effects.
9. **Profile and Optimize:** Procedural noise, especially complex multi-octave or 3D/4D variants, can be computationally intensive. Profile noise generation routines and consider optimizations such as pre-computation for static elements, GPU acceleration for visual aspects if feasible, or level-of-detail approaches for noise evaluation.

By strategically incorporating this broader palette of noise algorithms and leveraging their unique strengths for specific components of the world generation pipeline, the project can achieve a higher degree of realism, complexity, and emergent behavior in its procedurally generated worlds. The focus should be on how these different noise-driven systems interact to create a cohesive and believable whole.


#### 

---
Works cited



1. Procedural generation - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Procedural_generation](https://en.wikipedia.org/wiki/Procedural_generation)
2. Diving Into Procedural Content Generation, With WorldEngine - Smashing Magazine, accessed May 17, 2025, [https://www.smashingmagazine.com/2016/03/procedural-content-generation-introduction/](https://www.smashingmagazine.com/2016/03/procedural-content-generation-introduction/)
3. Perlin Noise: Implementation, Procedural Generation, and Simplex Noise - GarageFarm, accessed May 17, 2025, [https://garagefarm.net/blog/perlin-noise-implementation-procedural-generation-and-simplex-noise](https://garagefarm.net/blog/perlin-noise-implementation-procedural-generation-and-simplex-noise)
4. Perlin noise (article) | Noise - Khan Academy, accessed May 17, 2025, [https://www.khanacademy.org/computing/computer-programming/programming-natural-simulations/programming-noise/a/perlin-noise](https://www.khanacademy.org/computing/computer-programming/programming-natural-simulations/programming-noise/a/perlin-noise)
5. Noise Pattern Generation: Techniques, Applications, and Benefits - FuryPage, accessed May 17, 2025, [https://furypage.com/blog/noise-pattern-generation](https://furypage.com/blog/noise-pattern-generation)
6. Procedurally Generating Terrain, accessed May 17, 2025, [https://micsymposium.org/mics_2011_proceedings/mics2011_submission_30.pdf](https://micsymposium.org/mics_2011_proceedings/mics2011_submission_30.pdf)
7. Unity Terrain Generation, accessed May 17, 2025, [https://ckempke.github.io/UnityTerrainGeneration/generation/](https://ckempke.github.io/UnityTerrainGeneration/generation/)
8. Survey of Procedural Methods for Two-Dimensional Texture Generation - PMC, accessed May 17, 2025, [https://pmc.ncbi.nlm.nih.gov/articles/PMC7070409/](https://pmc.ncbi.nlm.nih.gov/articles/PMC7070409/)
9. Worley noise - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Worley_noise](https://en.wikipedia.org/wiki/Worley_noise)
10. www.cs.ubc.ca, accessed May 17, 2025, [https://www.cs.ubc.ca/~rbridson/docs/bridson-siggraph2007-curlnoise.pdf](https://www.cs.ubc.ca/~rbridson/docs/bridson-siggraph2007-curlnoise.pdf)
11. Perlin Noise: A Procedural Generation Algorithm - Raouf's blog, accessed May 17, 2025, [https://rtouti.github.io/graphics/perlin-noise-algorithm](https://rtouti.github.io/graphics/perlin-noise-algorithm)
12. How to generate a Wind Field using perlin noise - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/generative/comments/f6vlg5/how_to_generate_a_wind_field_using_perlin_noise/](https://www.reddit.com/r/generative/comments/f6vlg5/how_to_generate_a_wind_field_using_perlin_noise/)
13. Mountains, Cliffs, and Caves: A Guide to Using Perlin Noise for Procedural Gen, accessed May 17, 2025, [https://news.ycombinator.com/item?id=43257506](https://news.ycombinator.com/item?id=43257506)
14. Procedural World Gen with Plate Tectonics | Details | Hackaday.io, accessed May 17, 2025, [https://hackaday.io/project/196161-arduino-minecraft/log/229957-procedural-world-gen-with-plate-tectonics](https://hackaday.io/project/196161-arduino-minecraft/log/229957-procedural-world-gen-with-plate-tectonics)
15. Procedural World Generation - DTWorldz, accessed May 17, 2025, [https://dtworldz.hashnode.dev/procedural-world-generation](https://dtworldz.hashnode.dev/procedural-world-generation)
16. Asset - Project - Procedural Terrain Generation - GameMaker Community, accessed May 17, 2025, [https://forum.gamemaker.io/index.php?threads/procedural-terrain-generation.24896/](https://forum.gamemaker.io/index.php?threads/procedural-terrain-generation.24896/)
17. World generation using tectonic plate simulation and perlin noise. : r/proceduralgeneration, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/1fub6vw/world_generation_using_tectonic_plate_simulation/](https://www.reddit.com/r/proceduralgeneration/comments/1fub6vw/world_generation_using_tectonic_plate_simulation/)
18. Simplex noise - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Simplex_noise](https://en.wikipedia.org/wiki/Simplex_noise)
19. Simplex noise demystified - ITN, accessed May 17, 2025, [https://itn-web.it.liu.se/~stegu76/TNM084-2011/simplexnoise-demystified.pdf](https://itn-web.it.liu.se/~stegu76/TNM084-2011/simplexnoise-demystified.pdf)
20. Value noise - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Value_noise](https://en.wikipedia.org/wiki/Value_noise)
21. Noise Functions | gameidea, accessed May 17, 2025, [https://gameidea.org/2023/12/16/noise-functions/](https://gameidea.org/2023/12/16/noise-functions/)
22. Worley (cell) noise generator — noise_worley - ambient, accessed May 17, 2025, [https://ambient.data-imaginist.com/reference/noise_worley.html](https://ambient.data-imaginist.com/reference/noise_worley.html)
23. Math: F1, F2, F3 + noise - Filter Forge, accessed May 17, 2025, [https://www.filterforge.com/forum/read.php?FID=17&TID=7344](https://www.filterforge.com/forum/read.php?FID=17&TID=7344)
24. GLSL noise library: cellular noise variants - Red Blob Games, accessed May 17, 2025, [https://www.redblobgames.com/x/2107-webgl-noise/webgl-noise/webdemo/cellular.html](https://www.redblobgames.com/x/2107-webgl-noise/webgl-noise/webdemo/cellular.html)
25. Procedural Patterns And Noises - Neil Blevins, accessed May 17, 2025, [http://neilblevins.com/art_lessons/procedural_noise/procedural_noise.html](http://neilblevins.com/art_lessons/procedural_noise/procedural_noise.html)
26. FastNoiseLite — Godot Engine (stable) documentation in English, accessed May 17, 2025, [https://docs.godotengine.org/en/stable/classes/class_fastnoiselite.html](https://docs.godotengine.org/en/stable/classes/class_fastnoiselite.html)
27. Hello Worley - Procedural World, accessed May 17, 2025, [http://procworld.blogspot.com/2011/05/hello-worley.html](http://procworld.blogspot.com/2011/05/hello-worley.html)
28. Realtime planetary tectonics simulation : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/10as9mo/realtime_planetary_tectonics_simulation/](https://www.reddit.com/r/proceduralgeneration/comments/10as9mo/realtime_planetary_tectonics_simulation/)
29. Fundamentals of Terrain Generation - CMU School of Computer Science, accessed May 17, 2025, [https://www.cs.cmu.edu/~112-s23/notes/student-tp-guides/Terrain.pdf](https://www.cs.cmu.edu/~112-s23/notes/student-tp-guides/Terrain.pdf)
30. How to get a "noise map" like perlin noise, but with multiple distinct colored regions? : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/gjphdx/how_to_get_a_noise_map_like_perlin_noise_but_with/](https://www.reddit.com/r/proceduralgeneration/comments/gjphdx/how_to_get_a_noise_map_like_perlin_noise_but_with/)
31. Value Noise and Procedural Patterns - Scratchapixel, accessed May 17, 2025, [https://www.scratchapixel.com/lessons/procedural-generation-virtual-worlds/procedural-patterns-noise-part-1/simple-pattern-examples.html](https://www.scratchapixel.com/lessons/procedural-generation-virtual-worlds/procedural-patterns-noise-part-1/simple-pattern-examples.html)
32. Fractal Brownian Motion - The Book of Shaders, accessed May 17, 2025, [https://thebookofshaders.com/13/](https://thebookofshaders.com/13/)
33. 2 Procedural Fractal Terrains - Department of Computer Science, accessed May 17, 2025, [https://www.classes.cs.uchicago.edu/archive/2015/fall/23700-1/final-project/MusgraveTerrain00.pdf](https://www.classes.cs.uchicago.edu/archive/2015/fall/23700-1/final-project/MusgraveTerrain00.pdf)
34. www.sbgames.org, accessed May 17, 2025, [https://www.sbgames.org/sbgames2018/files/papers/ComputacaoShort/188264.pdf](https://www.sbgames.org/sbgames2018/files/papers/ComputacaoShort/188264.pdf)
35. Procedural Noise/Categories - PhysBAM, accessed May 17, 2025, [https://physbam.stanford.edu/cs448x/old/Procedural_Noise(2f)Categories.html](https://physbam.stanford.edu/cs448x/old/Procedural_Noise(2f)Categories.html)
36. Sampling Gabor Noise in the Spatial Domain - Computer Graphics | TU Wien, accessed May 17, 2025, [https://www.cg.tuwien.ac.at/research/publications/2014/charpenay-2014-sgn/charpenay-2014-sgn-paper.pdf](https://www.cg.tuwien.ac.at/research/publications/2014/charpenay-2014-sgn/charpenay-2014-sgn-paper.pdf)
37. Gabor Noise - Foundry Learn, accessed May 17, 2025, [https://learn.foundry.com/modo/content/help/pages/shading_lighting/shader_items/gabor_noise.html](https://learn.foundry.com/modo/content/help/pages/shading_lighting/shader_items/gabor_noise.html)
38. How to Use Gabor Filters to Generate Features for Machine Learning - Baeldung, accessed May 17, 2025, [https://www.baeldung.com/cs/ml-gabor-filters](https://www.baeldung.com/cs/ml-gabor-filters)
39. Procedural Noise using Sparse Gabor Convolution, accessed May 17, 2025, [https://graphics.cs.kuleuven.be/publications/LLDD09PNSGC/](https://graphics.cs.kuleuven.be/publications/LLDD09PNSGC/)
40. Gabor Noise - Jens Kafitz, accessed May 17, 2025, [https://campi3d.com/External/MariExtensionPack/userGuide5R8/GaborNoise.html](https://campi3d.com/External/MariExtensionPack/userGuide5R8/GaborNoise.html)
41. Simulating the structure and texture of solid wood - Cornell Computer Science, accessed May 17, 2025, [https://www.cs.cornell.edu/projects/wood/simulating_the_structure_and_texture_of_solid_wood.pdf](https://www.cs.cornell.edu/projects/wood/simulating_the_structure_and_texture_of_solid_wood.pdf)
42. Gabor Texture Node - Blender 4.4 Manual, accessed May 17, 2025, [https://docs.blender.org/manual/en/latest/render/shader_nodes/textures/gabor.html](https://docs.blender.org/manual/en/latest/render/shader_nodes/textures/gabor.html)
43. Gabor Noise by Example, accessed May 17, 2025, [https://graphics.cs.kuleuven.be/publications/GLLD12GNBE/GLLD12GNBE_paper.pdf](https://graphics.cs.kuleuven.be/publications/GLLD12GNBE/GLLD12GNBE_paper.pdf)
44. Improving Gabor Noise - Inria, accessed May 17, 2025, [https://www-sop.inria.fr/reves/Basilic/2011/LLD11/paper.pdf](https://www-sop.inria.fr/reves/Basilic/2011/LLD11/paper.pdf)
45. (PDF) Gabor Noise by Example - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/254461083_Gabor_Noise_by_Example](https://www.researchgate.net/publication/254461083_Gabor_Noise_by_Example)
46. Procedural Noise using Sparse Gabor Convolution - Auxiliary Material | Request PDF, accessed May 17, 2025, [https://www.researchgate.net/publication/277293838_Procedural_Noise_using_Sparse_Gabor_Convolution_-_Auxiliary_Material](https://www.researchgate.net/publication/277293838_Procedural_Noise_using_Sparse_Gabor_Convolution_-_Auxiliary_Material)
47. Wavelet Noise - Pixar Graphics, accessed May 17, 2025, [https://graphics.pixar.com/library/WaveletNoise/paper.pdf](https://graphics.pixar.com/library/WaveletNoise/paper.pdf)
48. Terrain synthesis using noise - CORE, accessed May 17, 2025, [https://core.ac.uk/download/pdf/250147208.pdf](https://core.ac.uk/download/pdf/250147208.pdf)
49. [Unity] Procedural Planets (E03: layered noise) - YouTube, accessed May 17, 2025, [https://m.youtube.com/watch?v=uY9PAcNMu8s](https://m.youtube.com/watch?v=uY9PAcNMu8s)
50. Making maps with noise functions - Red Blob Games, accessed May 17, 2025, [https://www.redblobgames.com/maps/terrain-from-noise/](https://www.redblobgames.com/maps/terrain-from-noise/)
51. "Procedural Generation of Terrain" - Suzanne Baxter (PyConline AU 2020) - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=GbEBMNoRfEU](https://www.youtube.com/watch?v=GbEBMNoRfEU)
52. [Unity] Procedural Planets (E03: layered noise) - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=uY9PAcNMu8s](https://www.youtube.com/watch?v=uY9PAcNMu8s)
53. (PDF) Polynomial method for Procedural Terrain Generation - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/309037528_Polynomial_method_for_Procedural_Terrain_Generation](https://www.researchgate.net/publication/309037528_Polynomial_method_for_Procedural_Terrain_Generation)
54. Hardware-Accelerated Gradient Noise for Graphics, accessed May 17, 2025, [https://www.eng.utah.edu/~cs6958/papers/noise.pdf](https://www.eng.utah.edu/~cs6958/papers/noise.pdf)
55. Create a Curl Noise Effect | Learn, accessed May 17, 2025, [https://effecthouse.tiktok.com/learn/guides/visual-effects-editor/curl-noise-effect](https://effecthouse.tiktok.com/learn/guides/visual-effects-editor/curl-noise-effect)
56. Curl Noise - al-ro, accessed May 17, 2025, [https://al-ro.github.io/projects/curl/](https://al-ro.github.io/projects/curl/)
57. Curl-noise for procedural fluid flow | Request PDF - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/216813629_Curl-noise_for_procedural_fluid_flow](https://www.researchgate.net/publication/216813629_Curl-noise_for_procedural_fluid_flow)
58. Curl Noise - GitHub, accessed May 17, 2025, [https://raw.githubusercontent.com/petewerner/misc/master/Curl%20Noise%20Slides.pdf](https://raw.githubusercontent.com/petewerner/misc/master/Curl%20Noise%20Slides.pdf)
59. Tiling Simplex Noise and Flow Noise in Two and Three Dimensions - Journal of Computer Graphics Techniques, accessed May 17, 2025, [https://jcgt.org/published/0011/01/02/paper.pdf](https://jcgt.org/published/0011/01/02/paper.pdf)
60. jcgt.org, accessed May 17, 2025, [https://jcgt.org/published/0011/01/02/psrdnoise-supplement.pdf](https://jcgt.org/published/0011/01/02/psrdnoise-supplement.pdf)
61. MIT Open Access Articles Anisotropic noise, accessed May 17, 2025, [https://dspace.mit.edu/bitstream/handle/1721.1/100393/AnisotropicNoise.pdf;jsessionid=50FC5FBDF5565687556369D0365D11BC?sequence=1](https://dspace.mit.edu/bitstream/handle/1721.1/100393/AnisotropicNoise.pdf;jsessionid=50FC5FBDF5565687556369D0365D11BC?sequence=1)
62. Anisotropic Noise - CORE, accessed May 17, 2025, [https://core.ac.uk/download/pdf/78065800.pdf](https://core.ac.uk/download/pdf/78065800.pdf)
63. Gabor Noise by Example - Inria, accessed May 17, 2025, [http://www-sop.inria.fr/reves/Basilic/2012/GLLD12/paper_0033_preprint.pdf](http://www-sop.inria.fr/reves/Basilic/2012/GLLD12/paper_0033_preprint.pdf)
64. dandrino/terrain-erosion-3-ways: Three Ways of Generating Terrain with Erosion Features - GitHub, accessed May 17, 2025, [https://github.com/dandrino/terrain-erosion-3-ways](https://github.com/dandrino/terrain-erosion-3-ways)
65. Noises | World Creator, accessed May 17, 2025, [https://docs.world-creator.com/reference/terrain/noises](https://docs.world-creator.com/reference/terrain/noises)
66. Procedurally generated gas giant (using domain warping) : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/eoxi2o/procedurally_generated_gas_giant_using_domain/](https://www.reddit.com/r/proceduralgeneration/comments/eoxi2o/procedurally_generated_gas_giant_using_domain/)
67. 2D weather simulation with cellular automata - reasonable? : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/1htspwa/2d_weather_simulation_with_cellular_automata/](https://www.reddit.com/r/proceduralgeneration/comments/1htspwa/2d_weather_simulation_with_cellular_automata/)
68. Building Better Terrain - ThingOnItsOwn, accessed May 17, 2025, [http://thingonitsown.blogspot.com/2018/11/building-better-terrain.html](http://thingonitsown.blogspot.com/2018/11/building-better-terrain.html)
69. Multifractal system - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/Multifractal_system](https://en.wikipedia.org/wiki/Multifractal_system)
70. Multifractal emergent processes: Multiplicative interactions override nonlinear component properties - arXiv, accessed May 17, 2025, [https://arxiv.org/html/2401.05105v1](https://arxiv.org/html/2401.05105v1)
71. (Un)usual terrain generation : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/ol23ui/unusual_terrain_generation/](https://www.reddit.com/r/proceduralgeneration/comments/ol23ui/unusual_terrain_generation/)
72. Prime gradient noise - SciOpen, accessed May 17, 2025, [https://www.sciopen.com/article_pdf/1476364088811126785.pdf](https://www.sciopen.com/article_pdf/1476364088811126785.pdf)
73. How to best layer *different* types of Noise? Billowed, Ridged, Fractal, etc? - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/qljj87/how_to_best_layer_different_types_of_noise/](https://www.reddit.com/r/proceduralgeneration/comments/qljj87/how_to_best_layer_different_types_of_noise/)
74. Noise Procedural - Modo - Foundry Learn, accessed May 17, 2025, [https://learn.foundry.com/modo/current/content/help/pages/shading_lighting/shader_items/noise.html](https://learn.foundry.com/modo/current/content/help/pages/shading_lighting/shader_items/noise.html)
75. noise - Rust - Docs.rs, accessed May 17, 2025, [https://docs.rs/noise/latest/noise/](https://docs.rs/noise/latest/noise/)
76. Attribute Noise 2.0 geometry node - SideFX, accessed May 17, 2025, [https://www.sidefx.com/docs/houdini/nodes/sop/attribnoise.html](https://www.sidefx.com/docs/houdini/nodes/sop/attribnoise.html)
77. Value Noise and Procedural Patterns - Scratchapixel, accessed May 17, 2025, [https://www.scratchapixel.com/lessons/procedural-generation-virtual-worlds/procedural-patterns-noise-part-1/introduction.html](https://www.scratchapixel.com/lessons/procedural-generation-virtual-worlds/procedural-patterns-noise-part-1/introduction.html)
78. A hybrid approach to 3D procedural generation and manual creation. - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/2s62j2/a_hybrid_approach_to_3d_procedural_generation_and/](https://www.reddit.com/r/proceduralgeneration/comments/2s62j2/a_hybrid_approach_to_3d_procedural_generation_and/)
79. An Approach to Sound Synthesis with L-Systems | Nathan Ho, accessed May 17, 2025, [https://nathan.ho.name/posts/sound-synthesis-with-l-systems/](https://nathan.ho.name/posts/sound-synthesis-with-l-systems/)
80. Score generation with L−systems - Algorithmic Botany, accessed May 17, 2025, [https://algorithmicbotany.org/papers/score.icmc86.pdf](https://algorithmicbotany.org/papers/score.icmc86.pdf)
81. The PCG Paradox: How Procedural Generation Can Feel Repetitive and What to Do About It, accessed May 17, 2025, [https://www.wayline.io/blog/pcg-paradox-repetition-solutions](https://www.wayline.io/blog/pcg-paradox-repetition-solutions)
82. L-system - Wikipedia, accessed May 17, 2025, [https://en.wikipedia.org/wiki/L-system](https://en.wikipedia.org/wiki/L-system)
83. Procedural Generation Using L-Systems | gameidea, accessed May 17, 2025, [https://gameidea.org/2024/02/21/procedural-generation-using-l-systems/](https://gameidea.org/2024/02/21/procedural-generation-using-l-systems/)
84. fused attention mechanism-based ore sorting network - arXiv, accessed May 17, 2025, [https://arxiv.org/pdf/2405.02785](https://arxiv.org/pdf/2405.02785)
85. CADIA OPERATIONS NEW SOUTH WALES, AUSTRALIA NI 43-101 Technical Report - Newcrest Mining, accessed May 17, 2025, [https://www.newcrest.com/sites/default/files/2020-10/Technical%20Report%20on%20Cadia%20Operations%20as%20of%2030%20June%202020_0.pdf](https://www.newcrest.com/sites/default/files/2020-10/Technical%20Report%20on%20Cadia%20Operations%20as%20of%2030%20June%202020_0.pdf)
86. Chapter 1 Graphical modeling using L-systems - Algorithmic Botany, accessed May 17, 2025, [https://algorithmicbotany.org/papers/abop/abop-ch1.pdf](https://algorithmicbotany.org/papers/abop/abop-ch1.pdf)
87. An algorithm for generating vein images for realistic modeling of a leaf, accessed May 17, 2025, [https://www.cp.eng.chula.ac.th/~piak/paper/2002/cmm2002.pdf](https://www.cp.eng.chula.ac.th/~piak/paper/2002/cmm2002.pdf)
88. Scaling in branch thickness and the fractal aesthetic of trees | PNAS Nexus | Oxford Academic, accessed May 17, 2025, [https://academic.oup.com/pnasnexus/article/4/2/pgaf003/7996468](https://academic.oup.com/pnasnexus/article/4/2/pgaf003/7996468)
89. Procedural Plant Generation with L-Systems - YouTube, accessed May 17, 2025, [https://m.youtube.com/watch?v=feNVBEPXAcE&t=361s](https://m.youtube.com/watch?v=feNVBEPXAcE&t=361s)
90. Introduction to L Systems: Generating Procedural Plants - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=3Mu0--aGfqg](https://www.youtube.com/watch?v=3Mu0--aGfqg)
91. Spot Noise - EMSOPEDIA, accessed May 17, 2025, [https://www.emsopedia.org/entries/spot-noise/](https://www.emsopedia.org/entries/spot-noise/)
92. First iteration of my tectonic plate simulation on a sphere (voronoi cells, soft body physics, and Kriging to sample heights at voronoi centroids instead of simulating every pixel) : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/1jhiznp/first_iteration_of_my_tectonic_plate_simulation/](https://www.reddit.com/r/proceduralgeneration/comments/1jhiznp/first_iteration_of_my_tectonic_plate_simulation/)
93. Procedural generation with Perlin noise variants - UPCommons, accessed May 17, 2025, [https://upcommons.upc.edu/bitstream/handle/2117/394002/160_Memoria_TFG.pdf;jsessionid=09FC43B76F9443F1C3BCE9A00E6532D0?sequence=2](https://upcommons.upc.edu/bitstream/handle/2117/394002/160_Memoria_TFG.pdf;jsessionid=09FC43B76F9443F1C3BCE9A00E6532D0?sequence=2)
94. Minecraft Procedural World Terrain Generation - STEAM News, accessed May 17, 2025, [https://www.steamnews.org/articles/technology/minecraft-procedural-world-terrain-generation](https://www.steamnews.org/articles/technology/minecraft-procedural-world-terrain-generation)
95. 3D terrain generation: geological slices and noise function, accessed May 17, 2025, [https://gamedev.stackexchange.com/questions/156940/3d-terrain-generation-geological-slices-and-noise-function](https://gamedev.stackexchange.com/questions/156940/3d-terrain-generation-geological-slices-and-noise-function)
96. 3D Procedural World Generation - gameidea, accessed May 17, 2025, [https://gameidea.org/2023/12/11/3d-procedural-world-generation/](https://gameidea.org/2023/12/11/3d-procedural-world-generation/)
97. Boundary Handling for Cohesive Tiling in Particle-Based Hydraulic Erosion Simulations - DiVA portal, accessed May 17, 2025, [http://www.diva-portal.org/smash/get/diva2:1868184/FULLTEXT01.pdf](http://www.diva-portal.org/smash/get/diva2:1868184/FULLTEXT01.pdf)
98. Making a Procedurally Generated Level for our RTS Game - gameidea, accessed May 17, 2025, [https://gameidea.org/2024/12/13/making-a-procedurally-generated-level-for-our-rts-game/](https://gameidea.org/2024/12/13/making-a-procedurally-generated-level-for-our-rts-game/)
99. Introduction to Procedural Generation with Perlin Noise for Game Development, accessed May 17, 2025, [https://www.gamegeniuslab.com/tutorial-post/introduction-to-procedural-generation-with-perlin-noise-for-game-development/](https://www.gamegeniuslab.com/tutorial-post/introduction-to-procedural-generation-with-perlin-noise-for-game-development/)
100. Procedural generation with Perlin noise variants - UPCommons, accessed May 17, 2025, [https://upcommons.upc.edu/bitstream/handle/2117/394002/160_Memoria_TFG.pdf;jsessionid=481E3DCC447321EB48022198F4FB19FA?sequence=2](https://upcommons.upc.edu/bitstream/handle/2117/394002/160_Memoria_TFG.pdf;jsessionid=481E3DCC447321EB48022198F4FB19FA?sequence=2)
101. Wavelet noise | Request PDF - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/220183597_Wavelet_noise](https://www.researchgate.net/publication/220183597_Wavelet_noise)
102. Procedural World Generation - GDC Vault, accessed May 17, 2025, [https://media.gdcvault.com/gdc2018/presentations/ProceduralWorldGeneration.pdf](https://media.gdcvault.com/gdc2018/presentations/ProceduralWorldGeneration.pdf)
103. Biome selection in procedurally-generated worlds : r/howdidtheycodeit - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/howdidtheycodeit/comments/17e2lmo/biome_selection_in_procedurallygenerated_worlds/](https://www.reddit.com/r/howdidtheycodeit/comments/17e2lmo/biome_selection_in_procedurallygenerated_worlds/)
104. Generating Procedural Maps using Perlin Noise - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=6BdYzfVOyBY](https://www.youtube.com/watch?v=6BdYzfVOyBY)
105. River Generation Algorithms : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/7v1y5l/river_generation_algorithms/](https://www.reddit.com/r/proceduralgeneration/comments/7v1y5l/river_generation_algorithms/)
106. Watershed Delineation from the Medial Axis of River Networks - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/254084470_Watershed_Delineation_from_the_Medial_Axis_of_River_Networks](https://www.researchgate.net/publication/254084470_Watershed_Delineation_from_the_Medial_Axis_of_River_Networks)
107. 7. SUMMARY, CONCLUSIONS, AND RECOMMENDATIONS, accessed May 17, 2025, [https://www.caee.utexas.edu/prof/maidment/GISHYDRO/pawel/midwest/conclus.pdf](https://www.caee.utexas.edu/prof/maidment/GISHYDRO/pawel/midwest/conclus.pdf)
108. Procedural Biome Generation: Ooze-Style - PROCJAM Tutorials, accessed May 17, 2025, [https://www.procjam.com/tutorials/en/ooze/](https://www.procjam.com/tutorials/en/ooze/)
109. Detecting Signals from Data with Noise: Theory and Applications - American Meteorological Society, accessed May 17, 2025, [https://journals.ametsoc.org/view/journals/atsc/70/5/jas-d-12-0213.1.pdf](https://journals.ametsoc.org/view/journals/atsc/70/5/jas-d-12-0213.1.pdf)
110. Solar radiation modification: evidence review report - Scientific Advice Mechanism, accessed May 17, 2025, [https://scientificadvice.eu/scientific-outputs/solar-radiation-modification-evidence-review-report/](https://scientificadvice.eu/scientific-outputs/solar-radiation-modification-evidence-review-report/)
111. Towards variance-conserving reconstructions of climate indices with Gaussian process regression in an embedding space - GMD, accessed May 17, 2025, [https://gmd.copernicus.org/articles/17/1765/](https://gmd.copernicus.org/articles/17/1765/)
112. Evaluating and Quantifying the Climate-Driven Interannual Variability in Global Inventory Modeling and Mapping Studies (GIMMS) Normalized Difference Vegetation Index (NDVI3g) at Global Scales - MDPI, accessed May 17, 2025, [https://www.mdpi.com/2072-4292/5/8/3918](https://www.mdpi.com/2072-4292/5/8/3918)
113. Procedural map generation using Perlin noise. How to improve? - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/16jcsu3/procedural_map_generation_using_perlin_noise_how/](https://www.reddit.com/r/proceduralgeneration/comments/16jcsu3/procedural_map_generation_using_perlin_noise_how/)
114. (PDF) Procedural Cloudscapes - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/323824245_Procedural_Cloudscapes](https://www.researchgate.net/publication/323824245_Procedural_Cloudscapes)
115. Title: Simulation of Cloud Formation using MAYA based particles, accessed May 17, 2025, [https://engineering.purdue.edu/~ebertd/ruchi1/report1.htm](https://engineering.purdue.edu/~ebertd/ruchi1/report1.htm)
116. Noise Functions - Texas Computer Science, accessed May 17, 2025, [https://www.cs.utexas.edu/~theshark/courses/cs354/lectures/cs354-21.pdf](https://www.cs.utexas.edu/~theshark/courses/cs354/lectures/cs354-21.pdf)
117. biome blending using multiple biome (altitude, humidity) points - Game Development Stack Exchange, accessed May 17, 2025, [https://gamedev.stackexchange.com/questions/208485/biome-blending-using-multiple-biome-altitude-humidity-points](https://gamedev.stackexchange.com/questions/208485/biome-blending-using-multiple-biome-altitude-humidity-points)
118. How could I use cellular noise to generate biomes? : r/Unity2D - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/Unity2D/comments/1fjh4z9/how_could_i_use_cellular_noise_to_generate_biomes/](https://www.reddit.com/r/Unity2D/comments/1fjh4z9/how_could_i_use_cellular_noise_to_generate_biomes/)
119. How to smooth between two perlin noise maps? : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/1g5yv2n/how_to_smooth_between_two_perlin_noise_maps/](https://www.reddit.com/r/proceduralgeneration/comments/1g5yv2n/how_to_smooth_between_two_perlin_noise_maps/)
120. 2D Wavelet Decomposition and F-K Migration for Identifying Fractured Rock Areas Using Ground Penetrating Radar - MDPI, accessed May 17, 2025, [https://www.mdpi.com/2072-4292/13/12/2280](https://www.mdpi.com/2072-4292/13/12/2280)
121. 3D Worley Noise | Substance 3D Designer - Adobe Help Center, accessed May 17, 2025, [https://helpx.adobe.com/substance-3d-designer/substance-compositing-graphs/nodes-reference-for-substance-compositing-graphs/node-library/texture-generators/noises/3d-worley-noise.html](https://helpx.adobe.com/substance-3d-designer/substance-compositing-graphs/nodes-reference-for-substance-compositing-graphs/node-library/texture-generators/noises/3d-worley-noise.html)
122. Removing multiple types of noise of distributed acoustic sensing seismic data using attention-guided denoising convolutional neural network - Frontiers, accessed May 17, 2025, [https://www.frontiersin.org/journals/earth-science/articles/10.3389/feart.2022.986470/full](https://www.frontiersin.org/journals/earth-science/articles/10.3389/feart.2022.986470/full)
123. Use of Wavelet Transformation for Geophysical Well-Log Data Analysis - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/224719018_Use_of_Wavelet_Transformation_for_Geophysical_Well-Log_Data_Analysis](https://www.researchgate.net/publication/224719018_Use_of_Wavelet_Transformation_for_Geophysical_Well-Log_Data_Analysis)
124. Coding Adventure: Hydraulic Erosion - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=eaXk97ujbPQ](https://www.youtube.com/watch?v=eaXk97ujbPQ)
125. Procedural Rock Generator with procedural Rock Shader - Blender Geometry Nodes Tutorial - YouTube, accessed May 17, 2025, [https://m.youtube.com/watch?v=y3OUFZI4HTI](https://m.youtube.com/watch?v=y3OUFZI4HTI)
126. www.lirmm.fr, accessed May 17, 2025, [https://www.lirmm.fr/~nfaraj/publications/flexible_erosion/2024_Flexible_Terrain_Erosion.pdf](https://www.lirmm.fr/~nfaraj/publications/flexible_erosion/2024_Flexible_Terrain_Erosion.pdf)
127. Erosion - QuadSpinner - Gaea Documentation, accessed May 17, 2025, [https://docs.quadspinner.com/Reference/Erosion/Erosion.html](https://docs.quadspinner.com/Reference/Erosion/Erosion.html)
128. The color of environmental noise in river networks - PMC - PubMed Central, accessed May 17, 2025, [https://pmc.ncbi.nlm.nih.gov/articles/PMC10050181/](https://pmc.ncbi.nlm.nih.gov/articles/PMC10050181/)
129. Procedural Rivers And Hills Using Worley Noise : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/3le5fg/procedural_rivers_and_hills_using_worley_noise/](https://www.reddit.com/r/proceduralgeneration/comments/3le5fg/procedural_rivers_and_hills_using_worley_noise/)
130. generating veins of ore : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/b5tj1x/generating_veins_of_ore/](https://www.reddit.com/r/proceduralgeneration/comments/b5tj1x/generating_veins_of_ore/)
131. How to get started with Dwarf Fortress-styled procedural generation? - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/l8ttnq/how_to_get_started_with_dwarf_fortressstyled/](https://www.reddit.com/r/proceduralgeneration/comments/l8ttnq/how_to_get_started_with_dwarf_fortressstyled/)
132. How do they procedurally generate ore veins in Minecraft and Terraria - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/howdidtheycodeit/comments/myqmd7/how_do_they_procedurally_generate_ore_veins_in/](https://www.reddit.com/r/howdidtheycodeit/comments/myqmd7/how_do_they_procedurally_generate_ore_veins_in/)
133. Evaluation of Deep Isolation Forest (DIF) Algorithm for Mineral Prospectivity Mapping of Polymetallic Deposits - MDPI, accessed May 17, 2025, [https://www.mdpi.com/2075-163X/14/10/1015](https://www.mdpi.com/2075-163X/14/10/1015)
134. The range of ore bodies interpreted from 3D inversion results. - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/figure/The-range-of-ore-bodies-interpreted-from-3D-inversion-results_fig9_343828182](https://www.researchgate.net/figure/The-range-of-ore-bodies-interpreted-from-3D-inversion-results_fig9_343828182)
135. Different Kinds of 3D Noise? : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/19elmex/different_kinds_of_3d_noise/](https://www.reddit.com/r/proceduralgeneration/comments/19elmex/different_kinds_of_3d_noise/)
136. How to Generate Rock Features using Noises - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=iv_Ldwwkkbg](https://www.youtube.com/watch?v=iv_Ldwwkkbg)
137. Cloud Billowy Noise geometry node - SideFX, accessed May 17, 2025, [https://www.sidefx.com/docs/houdini/nodes/sop/cloudbillowynoise.html](https://www.sidefx.com/docs/houdini/nodes/sop/cloudbillowynoise.html)
138. How to Generate 3D Terrain in Scratch Using Perlin Noise - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=Q7a3E-D0gBg](https://www.youtube.com/watch?v=Q7a3E-D0gBg)
139. Coding Challenge 11: 3D Terrain Generation with Perlin Noise in Processing - YouTube, accessed May 17, 2025, [https://m.youtube.com/watch?v=IKB1hWWedMk&t=0s](https://m.youtube.com/watch?v=IKB1hWWedMk&t=0s)
140. [2504.11003] 3D Gabor Splatting: Reconstruction of High-frequency Surface Texture using Gabor Noise - arXiv, accessed May 17, 2025, [https://arxiv.org/abs/2504.11003](https://arxiv.org/abs/2504.11003)
141. GEO 2000 Abstracts | GeoArabia - GeoScienceWorld, accessed May 17, 2025, [https://pubs.geoscienceworld.org/gpl/geoarabia/article/5/1/8/566627/GEO-2000-Abstracts](https://pubs.geoscienceworld.org/gpl/geoarabia/article/5/1/8/566627/GEO-2000-Abstracts)
142. Rman Collection | LightWave, accessed May 17, 2025, [https://docs.lightwave3d.com/2025/rman-collection.html](https://docs.lightwave3d.com/2025/rman-collection.html)
143. asNoise2D — appleseed-maya Documentation, accessed May 17, 2025, [https://appleseed.readthedocs.io/projects/appleseed-maya/en/master/shaders/texture/as_noise2d.html](https://appleseed.readthedocs.io/projects/appleseed-maya/en/master/shaders/texture/as_noise2d.html)
144. Improving Gabor noise - PubMed, accessed May 17, 2025, [https://pubmed.ncbi.nlm.nih.gov/21041873/](https://pubmed.ncbi.nlm.nih.gov/21041873/)
145. Procedural Noise using Sparse Gabor Convolution - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=1_Ss2dUvaW8](https://www.youtube.com/watch?v=1_Ss2dUvaW8)
146. KEINOS/go-noise: Easy-to-use noise generator package in Golang for Perlin Noise and OpenSimplex Noise. - GitHub, accessed May 17, 2025, [https://github.com/KEINOS/go-noise](https://github.com/KEINOS/go-noise)
147. A Guide to the DBSCAN Clustering Algorithm - DataCamp, accessed May 17, 2025, [https://www.datacamp.com/tutorial/dbscan-clustering-algorithm](https://www.datacamp.com/tutorial/dbscan-clustering-algorithm)
148. DBSCAN - MATLAB & Simulink, accessed May 17, 2025, [https://www.mathworks.com/help/stats/dbscan-clustering.html](https://www.mathworks.com/help/stats/dbscan-clustering.html)
149. Advanced solar radiation prediction using combined satellite imagery and tabular data processing - PMC, accessed May 17, 2025, [https://pmc.ncbi.nlm.nih.gov/articles/PMC12018948/](https://pmc.ncbi.nlm.nih.gov/articles/PMC12018948/)
150. Density-Based Spatial Clustering of Application with Noise Algorithm for the Classification of Solar Radiation Time Series | Request PDF - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/310605487_Density-Based_Spatial_Clustering_of_Application_with_Noise_Algorithm_for_the_Classification_of_Solar_Radiation_Time_Series](https://www.researchgate.net/publication/310605487_Density-Based_Spatial_Clustering_of_Application_with_Noise_Algorithm_for_the_Classification_of_Solar_Radiation_Time_Series)
151. cjslep/noise: Noise generation library in Go - GitHub, accessed May 17, 2025, [https://github.com/cjslep/noise](https://github.com/cjslep/noise)
152. noise package - github.com/cjslep/noise - Go Packages, accessed May 17, 2025, [https://pkg.go.dev/github.com/cjslep/noise](https://pkg.go.dev/github.com/cjslep/noise)
153. A fabric shader using Gabor noise - Small Blender Things, accessed May 17, 2025, [https://blog.michelanders.nl/2013/02/a-fabric-shader-using-gabor-noise_77.html](https://blog.michelanders.nl/2013/02/a-fabric-shader-using-gabor-noise_77.html)
154. larspensjo/Go-simplex-noise - GitHub, accessed May 17, 2025, [https://github.com/larspensjo/Go-simplex-noise](https://github.com/larspensjo/Go-simplex-noise)
155. wavelet package - github.com/octu0/wavelet - Go Packages, accessed May 17, 2025, [https://pkg.go.dev/github.com/octu0/wavelet](https://pkg.go.dev/github.com/octu0/wavelet)
156. goccmack/godsp: Digital signal processing package in Go for the discrete wavelet transform (DWT) - GitHub, accessed May 17, 2025, [https://github.com/goccmack/godsp](https://github.com/goccmack/godsp)
157. Infinite Worley/Cellular/Voroni Noise : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/4u0s1i/infinite_worleycellularvoroni_noise/](https://www.reddit.com/r/proceduralgeneration/comments/4u0s1i/infinite_worleycellularvoroni_noise/)
158. Curl noise - how to get multi-scale vortices and my findings : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/18y4wco/curl_noise_how_to_get_multiscale_vortices_and_my/](https://www.reddit.com/r/proceduralgeneration/comments/18y4wco/curl_noise_how_to_get_multiscale_vortices_and_my/)
159. gopi-erabati/Wavelet-Transform-and-Application-to-Image-Denoising - GitHub, accessed May 17, 2025, [https://github.com/gopi-erabati/Wavelet-Transform-and-Application-to-Image-Denoising](https://github.com/gopi-erabati/Wavelet-Transform-and-Application-to-Image-Denoising)
160. Go framework/library similar to clojure's core.async.flow? : r/golang - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/golang/comments/1kaolbn/go_frameworklibrary_similar_to_clojures/](https://www.reddit.com/r/golang/comments/1kaolbn/go_frameworklibrary_similar_to_clojures/)
161. Unity.Mathematics/src/Unity.Mathematics/Noise/cellular2x2.cs at master · Unity-Technologies/Unity.Mathematics · GitHub, accessed May 17, 2025, [https://github.com/Unity-Technologies/Unity.Mathematics/blob/master/src/Unity.Mathematics/Noise/cellular2x2.cs](https://github.com/Unity-Technologies/Unity.Mathematics/blob/master/src/Unity.Mathematics/Noise/cellular2x2.cs)
162. ap-atul/wavelets: A simple and easy implementation of Wavelet Transform - GitHub, accessed May 17, 2025, [https://github.com/ap-atul/wavelets](https://github.com/ap-atul/wavelets)
163. flynn/noise: Go implementation of the Noise Protocol Framework - GitHub, accessed May 17, 2025, [https://github.com/flynn/noise](https://github.com/flynn/noise)
164. golang functions: parallel execution with return - Stack Overflow, accessed May 17, 2025, [https://stackoverflow.com/questions/27792389/golang-functions-parallel-execution-with-return](https://stackoverflow.com/questions/27792389/golang-functions-parallel-execution-with-return)
165. go.mod - perlin-network/wavelet - GitHub, accessed May 17, 2025, [https://github.com/perlin-network/wavelet/blob/master/go.mod](https://github.com/perlin-network/wavelet/blob/master/go.mod)
166. Erkaman/glsl-worley: Worley noise implementation for WebGL shaders - GitHub, accessed May 17, 2025, [https://github.com/Erkaman/glsl-worley](https://github.com/Erkaman/glsl-worley)
167. OpencV getGaborKernel() Method - GeeksforGeeks, accessed May 17, 2025, [https://www.geeksforgeeks.org/opencv-getgaborkernel-method/](https://www.geeksforgeeks.org/opencv-getgaborkernel-method/)
168. Noise Removing Technique in Computer Vision | GeeksforGeeks, accessed May 17, 2025, [https://www.geeksforgeeks.org/noise-removing-technique-in-computer-vision/](https://www.geeksforgeeks.org/noise-removing-technique-in-computer-vision/)
169. Auburn/FastNoiseLite: Fast Portable Noise Library - C# ... - GitHub, accessed May 17, 2025, [https://github.com/Auburn/FastNoiseLite](https://github.com/Auburn/FastNoiseLite)
170. Documentation · Auburn/FastNoiseLite Wiki - GitHub, accessed May 17, 2025, [https://github.com/Auburn/FastNoiseLite/wiki/Documentation](https://github.com/Auburn/FastNoiseLite/wiki/Documentation)
171. FastNoiseLite | Documentation - GitHub Pages, accessed May 17, 2025, [https://migueldeicaza.github.io/SwiftGodotDocs/documentation/swiftgodot/fastnoiselite/](https://migueldeicaza.github.io/SwiftGodotDocs/documentation/swiftgodot/fastnoiselite/)
172. Godot 4 - FastNoiseLite basics - YouTube, accessed May 17, 2025, [https://www.youtube.com/watch?v=dTdjgBvtC0E](https://www.youtube.com/watch?v=dTdjgBvtC0E)
173. noise - Rust - Docs.rs, accessed May 17, 2025, [https://docs.rs/noise](https://docs.rs/noise)
174. Razaekel/noise-rs: Procedural noise generation library for Rust. - GitHub, accessed May 17, 2025, [https://github.com/Razaekel/noise-rs](https://github.com/Razaekel/noise-rs)
175. libnoise - Rust, accessed May 17, 2025, [https://docs.rs/libnoise](https://docs.rs/libnoise)
176. Documentation - libnoise, accessed May 17, 2025, [https://libnoise.sourceforge.net/docs/](https://libnoise.sourceforge.net/docs/)
177. kbladin/Curl_Noise: Implementation of curl noise for particles simulated on GPU with OpenGL - GitHub, accessed May 17, 2025, [https://github.com/kbladin/Curl_Noise](https://github.com/kbladin/Curl_Noise)
178. webgl-gabor-noise/gnoise.glsl at master - GitHub, accessed May 17, 2025, [https://github.com/victor-shepardson/webgl-gabor-noise/blob/master/gnoise.glsl](https://github.com/victor-shepardson/webgl-gabor-noise/blob/master/gnoise.glsl)
179. Sparse (CPU) noise in GLSL - TouchDesigner forum, accessed May 17, 2025, [https://forum.derivative.ca/t/sparse-cpu-noise-in-glsl/499808](https://forum.derivative.ca/t/sparse-cpu-noise-in-glsl/499808)
180. Surrendering now - a plea for help understanding simplex noise implementation in glsl : r/proceduralgeneration - Reddit, accessed May 17, 2025, [https://www.reddit.com/r/proceduralgeneration/comments/mi25yg/surrendering_now_a_plea_for_help_understanding/](https://www.reddit.com/r/proceduralgeneration/comments/mi25yg/surrendering_now_a_plea_for_help_understanding/)
181. Getting the Most Out of Noise in UE4 - Unreal Engine, accessed May 17, 2025, [https://www.unrealengine.com/en-US/tech-blog/getting-the-most-out-of-noise-in-ue4](https://www.unrealengine.com/en-US/tech-blog/getting-the-most-out-of-noise-in-ue4)
182. (PDF) Sampling Gabor Noise in the Spatial Domain - ResearchGate, accessed May 17, 2025, [https://www.researchgate.net/publication/264451797_Sampling_Gabor_Noise_in_the_Spatial_Domain](https://www.researchgate.net/publication/264451797_Sampling_Gabor_Noise_in_the_Spatial_Domain)