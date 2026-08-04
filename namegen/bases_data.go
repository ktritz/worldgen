package namegen

// Code-generated data file (hand-checked). Do not edit the lexicons casually:
// the Markov chains and tests depend on them.
//
// The seed lexicons below are copied from Azgaar's Fantasy Map Generator
// name-bases data (src/data/name-bases.ts), MIT License,
// Copyright (c) Azgaar. https://github.com/Azgaar/Fantasy-Map-Generator
// Per-base min/max lengths and duplicable-letter sets are FMG's tuning values.

// baseSpec holds one name base: tuning parameters plus a seed lexicon of
// real-world or fantasy toponyms used to build the pseudo-syllable chain.
type baseSpec struct {
	Key     string // stable identifier used by the culture/ancestry mapping
	Min     int    // minimum generated name length (pre-cleanup, in bytes)
	Max     int    // maximum generated name length (pre-cleanup, in bytes)
	Dupl    string // letters allowed to appear doubled in output
	Lexicon string // comma-separated seed words
}

var baseSpecs = []baseSpec{
	{
		// German toponyms; merchant_league culture, human and gnome ancestries (FMG base "German")
		Key:  "germanic",
		Min:  5,
		Max:  12,
		Dupl: "lt",
		Lexicon: "Achern,Aichhalden,Aitern,Albbruck,Alpirsbach,Altensteig,Althengstett,Appenweier,Auggen,Badenen," +
			"Badenweiler,Baiersbronn,Ballrechten,Bellingen,Berghaupten,Bernau,Biberach,Biederbach,Binzen," +
			"Birkendorf,Birkenfeld,Bischweier,Blumberg,Bollen,Bollschweil,Bonndorf,Bosingen,Braunlingen," +
			"Breisach,Breisgau,Breitnau,Brigachtal,Buchenbach,Buggingen,Buhl,Buhlertal,Calw,Dachsberg,Dobel," +
			"Donaueschingen,Dornhan,Dornstetten,Dottingen,Dunningen,Durbach,Durrheim,Ebhausen,Ebringen," +
			"Efringen,Egenhausen,Ehrenkirchen,Ehrsberg,Eimeldingen,Eisenbach,Elzach,Elztal,Emmendingen," +
			"Endingen,Engelsbrand,Enz,Enzklosterle,Eschbronn,Ettenheim,Ettlingen,Feldberg,Fischerbach," +
			"Fischingen,Fluorn,Forbach,Freiamt,Freiburg,Freudenstadt,Friedenweiler,Friesenheim,Frohnd," +
			"Furtwangen,Gaggenau,Geisingen,Gengenbach,Gernsbach,Glatt,Glatten,Glottertal,Gorwihl,Gottenheim," +
			"Grafenhausen,Grenzach,Griesbach,Gutach,Gutenbach,Hag,Haiterbach,Hardt,Harmersbach,Hasel,Haslach," +
			"Hausach,Hausen,Hausern,Heitersheim,Herbolzheim,Herrenalb,Herrischried,Hinterzarten,Hochenschwand," +
			"Hofen,Hofstetten,Hohberg,Horb,Horben,Hornberg,Hufingen,Ibach,Ihringen,Inzlingen,Kandern,Kappel," +
			"Kappelrodeck,Karlsbad,Karlsruhe,Kehl,Keltern,Kippenheim,Kirchzarten,Konigsfeld,Krozingen," +
			"Kuppenheim,Kussaberg,Lahr,Lauchringen,Lauf,Laufenburg,Lautenbach,Lauterbach,Lenzkirch,Liebenzell," +
			"Loffenau,Loffingen,Lorrach,Lossburg,Mahlberg,Malsburg,Malsch,March,Marxzell,Marzell,Maulburg," +
			"Monchweiler,Muhlenbach,Mullheim,Munstertal,Murg,Nagold,Neubulach,Neuenburg,Neuhausen,Neuried," +
			"Neuweiler,Niedereschach,Nordrach,Oberharmersbach,Oberkirch,Oberndorf,Oberbach,Oberried," +
			"Oberwolfach,Offenburg,Ohlsbach,Oppenau,Ortenberg,otigheim,Ottenhofen,Ottersweier,Peterstal," +
			"Pfaffenweiler,Pfalzgrafenweiler,Pforzheim,Rastatt,Renchen,Rheinau,Rheinfelden,Rheinmunster," +
			"Rickenbach,Rippoldsau,Rohrdorf,Rottweil,Rummingen,Rust,Sackingen,Sasbach,Sasbachwalden," +
			"Schallbach,Schallstadt,Schapbach,Schenkenzell,Schiltach,Schliengen,Schluchsee,Schomberg,Schonach," +
			"Schonau,Schonenberg,Schonwald,Schopfheim,Schopfloch,Schramberg,Schuttertal,Schwenningen," +
			"Schworstadt,Seebach,Seelbach,Seewald,Sexau,Simmersfeld,Simonswald,Sinzheim,Solden,Staufen,Stegen," +
			"Steinach,Steinen,Steinmauern,Straubenhardt,Stuhlingen,Sulz,Sulzburg,Teinach,Tiefenbronn,Tiengen," +
			"Titisee,Todtmoos,Todtnau,Todtnauberg,Triberg,Tunau,Tuningen,uhlingen,Unterkirnach,Reichenbach," +
			"Utzenfeld,Villingen,Villingendorf,Vogtsburg,Vohrenbach,Waldachtal,Waldbronn,Waldkirch,Waldshut," +
			"Wehr,Weil,Weilheim,Weisenbach,Wembach,Wieden,Wiesental,Wildbad,Wildberg,Winzeln,Wittlingen," +
			"Wittnau,Wolfach,Wutach,Wutoschingen,Wyhlen,Zavelstein",
	},
	{
		// Icelandic/Norwegian toponyms; highland_hold culture (FMG base "Nordic")
		Key:  "nordic",
		Min:  6,
		Max:  10,
		Dupl: "kln",
		Lexicon: "Akureyri,Aldra,Alftanes,Andenes,Austbo,Auvog,Bakkafjordur,Ballangen,Bardal,Beisfjord,Bifrost," +
			"Bildudalur,Bjerka,Bjerkvik,Bjorkosen,Bliksvaer,Blokken,Blonduos,Bolga,Bolungarvik,Borg,Borgarnes," +
			"Bosmoen,Bostad,Bostrand,Botsvika,Brautarholt,Breiddalsvik,Bringsli,Brunahlid,Budardalur," +
			"Byggdakjarni,Dalvik,Djupivogur,Donnes,Drageid,Drangsnes,Egilsstadir,Eiteroga,Elvenes,Engavogen," +
			"Ertenvog,Eskifjordur,Evenes,Eyrarbakki,Fagernes,Fallmoen,Fellabaer,Fenes,Finnoya,Fjaer,Fjelldal," +
			"Flakstad,Flateyri,Flostrand,Fludir,Gardaber,Gardur,Gimstad,Givaer,Gjeroy,Gladstad,Godoya," +
			"Godoynes,Granmoen,Gravdal,Grenivik,Grimsey,Grindavik,Grytting,Hafnir,Halsa,Hauganes,Haugland," +
			"Hauknes,Hella,Helland,Hellissandur,Hestad,Higrav,Hnifsdalur,Hofn,Hofsos,Holand,Holar,Holen," +
			"Holkestad,Holmavik,Hopen,Hovden,Hrafnagil,Hrisey,Husavik,Husvik,Hvammstangi,Hvanneyri,Hveragerdi," +
			"Hvolsvollur,Igeroy,Indre,Inndyr,Innhavet,Innes,Isafjordur,Jarklaustur,Jarnsreykir,Junkerdal," +
			"Kaldvog,Kanstad,Karlsoy,Kavosen,Keflavik,Kjelde,Kjerstad,Klakk,Kopasker,Kopavogur,Korgen," +
			"Kristnes,Krutoga,Krystad,Kvina,Lande,Laugar,Laugaras,Laugarbakki,Laugarvatn,Laupstad,Leines," +
			"Leira,Leiren,Leland,Lenvika,Loding,Lodingen,Lonsbakki,Lopsmarka,Lovund,Luroy,Maela,Melahverfi," +
			"Meloy,Mevik,Misvaer,Mornes,Mosfellsber,Moskenes,Myken,Naurstad,Nesberg,Nesjahverfi,Nesset," +
			"Nevernes,Obygda,Ofoten,Ogskardet,Okervika,Oknes,Olafsfjordur,Oldervika,Olstad,Onstad,Oppeid," +
			"Oresvika,Orsnes,Orsvog,Osmyra,Overdal,Prestoya,Raudalaekur,Raufarhofn,Reipo,Reykholar,Reykholt," +
			"Reykjahlid,Rif,Rinoya,Rodoy,Rognan,Rosvika,Rovika,Salhus,Sanden,Sandgerdi,Sandoker,Sandset," +
			"Sandvika,Saudarkrokur,Selfoss,Selsoya,Sennesvik,Setso,Siglufjordur,Silvalen,Skagastrond," +
			"Skjerstad,Skonland,Skorvogen,Skrova,Sleneset,Snubba,Softing,Solheim,Solheimar,Sorarnoy,Sorfugloy," +
			"Sorland,Sormela,Sorvaer,Sovika,Stamsund,Stamsvika,Stave,Stokka,Stokkseyri,Storjord,Storo," +
			"Storvika,Strand,Straumen,Strendene,Sudavik,Sudureyri,Sundoya,Sydalen,Thingeyri,Thorlakshofn," +
			"Thorshofn,Tjarnabyggd,Tjotta,Tosbotn,Traelnes,Trofors,Trones,Tverro,Ulvsvog,Unnstad,Utskor,Valla," +
			"Vandved,Varmahlid,Vassos,Vevelstad,Vidrek,Vik,Vikholmen,Vogar,Vogehamn,Vopnafjordur",
	},
	{
		// Latin toponyms; imperial_mandate and iron_legion cultures (FMG base "Roman")
		Key:  "roman",
		Min:  6,
		Max:  11,
		Dupl: "ln",
		Lexicon: "Abila,Adflexum,Adnicrem,Aelia,Aelius,Aeminium,Aequum,Agrippina,Agrippinae,Ala,Albanianis,Aleria," +
			"Ambianum,Andautonia,Apulum,Aquae,Aquaegranni,Aquensis,Aquileia,Aquincum,Arae,Argentoratum," +
			"Ariminum,Ascrivium,Asturica,Atrebatum,Atuatuca,Augusta,Aurelia,Aurelianorum,Batavar,Batavorum," +
			"Belum,Biriciana,Blestium,Bonames,Bonna,Bononia,Borbetomagus,Bovium,Bracara,Brigantium,Burgodunum," +
			"Caesaraugusta,Caesarea,Caesaromagus,Calleva,Camulodunum,Cannstatt,Cantiacorum,Capitolina,Caralis," +
			"Castellum,Castra,Castrum,Cibalae,Clausentum,Colonia,Concangis,Condate,Confluentes,Conimbriga," +
			"Corduba,Coria,Corieltauvorum,Corinium,Coriovallum,Cornoviorum,Danum,Deva,Dianium,Divodurum," +
			"Dobunnorum,Drusi,Dubris,Dumnoniorum,Durnovaria,Durocobrivis,Durocornovium,Duroliponte,Durovernum," +
			"Durovigutum,Eboracum,Ebusus,Edetanorum,Emerita,Emona,Emporiae,Euracini,Faventia,Flaviae," +
			"Florentia,Forum,Gerulata,Gerunda,Gesoscribate,Glevensium,Hadriani,Herculanea,Isca,Italica,Iulia," +
			"Iuliobrigensium,Iuvavum,Lactodurum,Lagentium,Lapurdum,Lauri,Legionis,Lemanis,Lentia,Lepidi," +
			"Letocetum,Lindinis,Lindum,Lixus,Londinium,Lopodunum,Lousonna,Lucus,Lugdunum,Luguvalium,Lutetia," +
			"Mancunium,Marsonia,Martius,Massa,Massilia,Matilo,Mattiacorum,Mediolanum,Mod,Mogontiacum," +
			"Moridunum,Mursa,Naissus,Nervia,Nida,Nigrum,Novaesium,Noviomagus,Olicana,Olisippo,Ovilava," +
			"Parisiorum,Partiscum,Paterna,Pistoria,Placentia,Pollentia,Pomaria,Pompeii,Pons,Portus,Praetoria," +
			"Praetorium,Pullum,Ragusium,Ratae,Raurica,Ravenna,Regina,Regium,Regulbium,Rigomagus,Roma,Romula," +
			"Rutupiae,Salassorum,Salernum,Salona,Scalabis,Segovia,Silurum,Sirmium,Siscia,Sorviodurum," +
			"Sumelocenna,Tarraco,Taurinorum,Theranda,Traiectum,Treverorum,Tungrorum,Turicum,Ulpia,Valentia," +
			"Venetiae,Venta,Verulamium,Vesontio,Vetera,Victoriae,Victrix,Villa,Viminacium,Vindelicorum," +
			"Vindobona,Vinovia,Viroconium",
	},
	{
		// Arabic toponyms; desert_caravan culture (FMG base "Arabic")
		Key:  "arabic",
		Min:  4,
		Max:  9,
		Dupl: "ae",
		Lexicon: "Abha,Ajman,Alabar,Alarjam,Alashraf,Alawali,Albawadi,Albirk,Aldhabiyah,Alduwaid,Alfareeq,Algayed," +
			"Alhazim,Alhrateem,Alhudaydah,Alhuwaya,Aljahra,Aljubail,Alkhafah,Alkhalas,Alkhawaneej,Alkhen," +
			"Alkhobar,Alkhuznah,Allisafah,Almshaykh,Almurjan,Almuwayh,Almuzaylif,Alnaheem,Alnashifah,Alqah," +
			"Alqouz,Alqurayyat,Alradha,Alraqmiah,Alsadyah,Alsafa,Alshagab,Alshuqaiq,Alsilaa,Althafeer," +
			"Alwasqah,Amaq,Amran,Annaseem,Aqbiyah,Arafat,Arar,Ardah,Asfan,Ashayrah,Askar,Ayaar,Aziziyah,Baesh," +
			"Bahrah,Balhaf,Banizayd,Bidiyah,Bisha,Biyatah,Buqhayq,Burayda,Dafiyat,Damad,Dammam,Dariyah,Dhafar," +
			"Dhahran,Dhalkut,Dhurma,Dibab,Doha,Dukhan,Duwaibah,Enaker,Fadhla,Fahaheel,Fanateer,Farasan,Fardah," +
			"Fujairah,Ghalilah,Ghar,Ghizlan,Ghomgyah,Ghran,Hadiyah,Haffah,Hajanbah,Hajrah,Haqqaq,Haradh,Hasar," +
			"Hawiyah,Hebaa,Hefar,Hijal,Husnah,Huwailat,Huwaitah,Irqah,Isharah,Ithrah,Jamalah,Jarab,Jareef," +
			"Jazan,Jeddah,Jiblah,Jihanah,Jilah,Jizan,Joraibah,Juban,Jumeirah,Kamaran,Keyad,Khab,Khaiybar," +
			"Khasab,Khathirah,Khawarah,Khulais,Kumzar,Limah,Linah,Madrak,Mahab,Mahalah,Makhtar,Mashwar," +
			"Masirah,Masliyah,Mastabah,Mazhar,Medina,Meeqat,Mirbah,Mokhtara,Muharraq,Muladdah,Musaykah," +
			"Mushayrif,Musrah,Mussafah,Nafhan,Najran,Nakhab,Nizwa,Oman,Qadah,Qalhat,Qamrah,Qasam,Qosmah," +
			"Qurain,Quriyat,Qurwa,Radaa,Rafha,Rahlah,Rakamah,Rasheedah,Rasmadrakah,Risabah,Rustaq,Ryadh," +
			"Sabtaljarah,Sadah,Safinah,Saham,Saihat,Salalah,Salmiya,Shabwah,Shalim,Shaqra,Sharjah,Sharurah," +
			"Shatifiyah,Shidah,Shihar,Shoqra,Shuwaq,Sibah,Sihmah,Sinaw,Sirwah,Sohar,Suhailah,Sulaibiya,Sunbah," +
			"Tabuk,Taif,Taqah,Tarif,Tharban,Thuqbah,Thuwal,Tubarjal,Turaif,Turbah,Tuwaiq,Ubar,Umaljerem," +
			"Urayarah,Urwah,Wabrah,Warbah,Yabreen,Yadamah,Yafur,Yarim,Yemen,Yiyallah,Zabid,Zahwah,Zallaq," +
			"Zinjibar,Zulumah",
	},
	{
		// Welsh/Brythonic toponyms; forest_enclave culture, halfling ancestry (FMG base "Celtic")
		Key:  "celtic",
		Min:  4,
		Max:  12,
		Dupl: "nld",
		Lexicon: "Aberaman,Aberangell,Aberarth,Aberavon,Aberbanc,Aberbargoed,Aberbeeg,Abercanaid,Abercarn," +
			"Abercastle,Abercegir,Abercraf,Abercregan,Abercych,Abercynon,Aberdare,Aberdaron,Aberdaugleddau," +
			"Aberdeen,Aberdulais,Aberdyfi,Aberedw,Abereiddy,Abererch,Abereron,Aberfan,Aberffraw,Aberffrwd," +
			"Abergavenny,Abergele,Aberglasslyn,Abergorlech,Abergwaun,Abergwesyn,Abergwili,Abergwynfi," +
			"Abergwyngregyn,Abergynolwyn,Aberhafesp,Aberhonddu,Aberkenfig,Aberllefenni,Abermain,Abermaw," +
			"Abermorddu,Abermule,Abernant,Aberpennar,Aberporth,Aberriw,Abersoch,Abersychan,Abertawe,Aberteifi," +
			"Aberthin,Abertillery,Abertridwr,Aberystwyth,Achininver,Afonhafren,Alisaha,Anfosadh,Antinbhearmor," +
			"Ardenna,Attacon,Banwen,Beira,Bhrura,Bleddfa,Boioduro,Bona,Boskyny,Boslowenpolbrogh,Boudobriga," +
			"Bravon,Brigant,Briganta,Briva,Brosnach,Caersws,Cambodunum,Cambra,Caracta,Catumagos,Centobriga," +
			"Ceredigion,Chalain,Chearbhallain,Chlasaigh,Chormaic,Cuileannach,Dinn,Diwa,Dubingen,Duibhidighe," +
			"Duro,Ebora,Ebruac,Eburodunum,Eccles,Egloskuri,Eighe,Eireann,Elerghi,Ferkunos,Fhlaithnin," +
			"Gallbhuaile,Genua,Ghrainnse,Gwyles,Heartsease,Hebron,Hordh,Inbhear,Inbhir,Inbhirair,Innerleithen," +
			"Innerleven,Innerwick,Inver,Inveraldie,Inverallan,Inveralmond,Inveramsay,Inveran,Inveraray," +
			"Inverarnan,Inverbervie,Inverclyde,Inverell,Inveresk,Inverfarigaig,Invergarry,Invergordon," +
			"Invergowrie,Inverhaddon,Inverkeilor,Inverkeithing,Inverkeithney,Inverkip,Inverleigh,Inverleith," +
			"Inverloch,Inverlochlarig,Inverlochy,Invermay,Invermoriston,Inverness,Inveroran,Invershin," +
			"Inversnaid,Invertrossachs,Inverugie,Inveruglas,Inverurie,Iubhrach,Karardhek,Kilninver,Kirkcaldy," +
			"Kirkintilloch,Krake,Lanngorrow,Latense,Leming,Lindomagos,Llanaber,Llandidiwg,Llandyrnog," +
			"Llanfarthyn,Llangadwaldr,Llansanwyr,Lochinver,Lugduno,Magoduro,Mheara,Monmouthshire,Nanshiryarth," +
			"Narann,Novioduno,Nowijonago,Octoduron,Penning,Pheofharain,Ponsmeur,Raithin,Ricomago,Rossinver," +
			"Salodurum,Seguia,Sentica,Theorsa,Tobargeal,Trealaw,Trefesgob,Trewedhenek,Trewythelan,Tuaisceart," +
			"Uige,Vitodurum,Windobona",
	},
	{
		// Old East Slavic toponyms; river_confederacy and marsh_clans cultures (FMG base "Ruthenian")
		Key:  "slavic",
		Min:  5,
		Max:  10,
		Dupl: "",
		Lexicon: "Belgorod,Beloberezhye,Belyi,Belz,Berestiy,Berezhets,Berezovets,Berezutsk,Bobruisk,Bolonets," +
			"Borisov,Borovsk,Bozhesk,Bratslav,Bryansk,Brynsk,Buryn,Byhov,Chechersk,Chemesov,Cheremosh,Cherlen," +
			"Chern,Chernigov,Chernitsa,Chernobyl,Chernogorod,Chertoryesk,Chetvertnia,Demyansk,Derevesk," +
			"Devyagoresk,Dichin,Dmitrov,Dorogobuch,Dorogobuzh,Drestvin,Drokov,Drutsk,Dubechin,Dubichi,Dubki," +
			"Dubkov,Dveren,Galich,Glebovo,Glinsk,Goloty,Gomiy,Gorodets,Gorodische,Gorodno,Gorohovets,Goroshin," +
			"Gorval,Goryshon,Holm,Horobor,Hoten,Hotin,Hotmyzhsk,Ilovech,Ivan,Izborsk,Izheslavl,Kamenets,Kanev," +
			"Karachev,Karna,Kavarna,Klechesk,Klyapech,Kolomyya,Kolyvan,Kopyl,Korec,Kornik,Korochunov,Korshev," +
			"Korsun,Koshkin,Kotelno,Kovyla,Kozelsk,Kozelsk,Kremenets,Krichev,Krylatsk,Ksniatin,Kulatsk,Kursk," +
			"Kursk,Lebedev,Lida,Logosko,Lomihvost,Loshesk,Loshichi,Lubech,Lubno,Lubutsk,Lutsk,Luchin,Luki," +
			"Lukoml,Luzha,Lvov,Mtsensk,Mdin,Medniki,Melecha,Merech,Meretsk,Mescherskoe,Meshkovsk,Metlitsk," +
			"Mezetsk,Mglin,Mihailov,Mikitin,Mikulino,Miloslavichi,Mogilev,Mologa,Moreva,Mosalsk,Moschiny," +
			"Mozyr,Mstislav,Mstislavets,Muravin,Nemech,Nemiza,Nerinsk,Nichan,Novgorod,Novogorodok,Obolichi," +
			"Obolensk,Obolensk,Oleshsk,Olgov,Omelnik,Opoka,Opoki,Oreshek,Orlets,Osechen,Oster,Ostrog,Ostrov," +
			"Perelai,Peremil,Peremyshl,Pererov,Peresechen,Perevitsk,Pereyaslav,Pinsk,Ples,Polotsk,Pronsk," +
			"Proposhesk,Punia,Putivl,Rechitsa,Rodno,Rogachev,Romanov,Romny,Roslavl,Rostislavl,Rostovets,Rsha," +
			"Ruza,Rybchesk,Rylsk,Rzhavesk,Rzhev,Rzhischev,Sambor,Serensk,Serensk,Serpeysk,Shilov,Shuya,Sinech," +
			"Sizhka,Skala,Slovensk,Slutsk,Smedin,Sneporod,Snitin,Snovsk,Sochevo,Sokolec,Starica,Starodub," +
			"Stepan,Sterzh,Streshin,Sutesk,Svinetsk,Svisloch,Terebovl,Ternov,Teshilov,Teterin,Tiversk," +
			"Torchevsk,Toropets,Torzhok,Tripolye,Trubchevsk,Tur,Turov,Usvyaty,Uteshkov,Vasilkov,Velil,Velye," +
			"Venev,Venicha,Verderev,Vereya,Veveresk,Viazma,Vidbesk,Vidychev,Voino,Volodimer,Volok,Volyn," +
			"Vorobesk,Voronich,Voronok,Vorotynsk,Vrev,Vruchiy,Vselug,Vyatichsk,Vyatka,Vyshegorod,Vyshgorod," +
			"Vysokoe,Yagniatin,Yaropolch,Yasenets,Yuryev,Yuryevets,Zaraysk,Zhitomel,Zholvazh,Zizhech,Zubkov," +
			"Zudechev,Zvenigorod",
	},
	{
		// Classical Greek toponyms; sky_aerie culture, aarakocra ancestry (FMG base "Greek")
		Key:  "greek",
		Min:  5,
		Max:  11,
		Dupl: "s",
		Lexicon: "Abdera,Acharnae,Aegae,Aegina,Agrinion,Aigosthena,Akragas,Akroinon,Akrotiri,Alalia,Alexandria," +
			"Amarynthos,Amaseia,Amphicaea,Amphigeneia,Amphipolis,Antipatrea,Antiochia,Apamea,Aphidna," +
			"Apollonia,Argos,Artemita,Argyropolis,Asklepios,Athenai,Athmonia,Bhrytos,Borysthenes,Brauron," +
			"Byblos,Byzantion,Bythinion,Calydon,Chamaizi,Chalcis,Chios,Cleona,Corcyra,Croton,Cyrene,Cythera," +
			"Decelea,Delos,Delphi,Dicaearchia,Didyma,Dion,Dioscurias,Dodona,Dorylaion,Elateia,Eleusis," +
			"Eleutherna,Emporion,Ephesos,Epidamnos,Epidauros,Epizephyrian,Erythrae,Eubea,Golgi,Gonnos," +
			"Gorgippia,Gournia,Gortyn,Gytion,Hagios,Halicarnassos,Heliopolis,Hellespontos,Heloros,Heraclea," +
			"Hierapolis,Himera,Histria,Hubla,Hyele,Ialysos,Iasos,Idalion,Imbros,Iolcos,Itanos,Ithaca,Juktas," +
			"Kallipolis,Kameiros,Karistos,Kasmenai,Kepoi,Kimmerikon,Knossos,Korinthos,Kos,Kourion,Kydonia," +
			"Kyrenia,Lamia,Lampsacos,Laodicea,Lapithos,Larissa,Lebena,Lefkada,Lekhaion,Leibethra,Leontinoi," +
			"Lilaea,Lindos,Lissos,Magnesia,Mantineia,Marathon,Marmara,Massalia,Megalopolis,Megara,Metapontion," +
			"Methumna,Miletos,Morgantina,Mulai,Mukenai,Myonia,Myra,Myrmekion,Myos,Nauplios,Naucratis," +
			"Naupaktos,Naxos,Neapolis,Nemea,Nicaea,Nicopolis,Nymphaion,Nysa,Odessos,Olbia,Olympia,Olynthos," +
			"Opos,Orchomenos,Oricos,Orestias,Oreos,Onchesmos,Pagasae,Palaikastro,Pandosia,Panticapaion,Paphos," +
			"Pargamon,Paros,Pegai,Pelion,Peiraies,Phaistos,Phaleron,Pharos,Pithekussa,Philippopolis,Phocaea," +
			"Pinara,Pisa,Pitane,Plataea,Poseidonia,Potidaea,Pseira,Psychro,Pteleos,Pydna,Pylos,Pyrgos,Rhamnos," +
			"Rhithymna,Rhypae,Rizinia,Rodos,Salamis,Samos,Skyllaion,Seleucia,Semasos,Sestos,Scidros,Sicyon," +
			"Sinope,Siris,Smyrna,Sozopolis,Sparta,Stagiros,Stratos,Stymphalos,Sybaris,Surakousai,Taras," +
			"Tanagra,Tanais,Tauromenion,Tegea,Temnos,Teos,Thapsos,Thassos,Thebai,Theodosia,Therma,Thespian," +
			"Thronion,Thoricos,Thurii,Thyreum,Thyria,Tithoraea,Tomis,Tragurion,Tripolis,Troliton,Troy," +
			"Tylissos,Tyros,Vathypetros,Zakynthos,Zakros",
	},
	{
		// FMG fantasy Elven base; elf ancestry (FMG base "Elven")
		Key:  "elven",
		Min:  6,
		Max:  12,
		Dupl: "lenmsrg",
		Lexicon: "Adrindest,Aethel,Afranthemar,Aiqua,Alari,Allanar,Almalian,Alora,Alyanasari,Alyelona,Alyran,Ammar," +
			"Anyndell,Arasari,Aren,Ashmebel,Aymlume,Bel-Didhel,Brinorion,Caelora,Chaulssad,Chaundra,Cyhmel," +
			"Cyrang,Dolarith,Dolonde,Draethe,Dranzan,Draugaust,E'ana,Eahil,Edhil,Eebel,Efranluma," +
			"Eld-Sinnocrin,Elelthyr,Ellanalin,Ellena,Ellorthond,Eltaesi,Elunore,Emyranserine,Entheas," +
			"Eriargond,Esari,Esath,Eserius,Eshsalin,Eshthalas,Evraland,Faellenor,Famelenora,Filranlean," +
			"Filsaqua,Gafetheas,Gaf Serine,Geliene,Gondorwin,Guallu,Haeth,Hanluna,Haulssad,Heloriath," +
			"Himlarien,Himliene,Hinnead,Hlinas,Hloireenil,Hluihei,Hlurthei,Hlynead,Iaenarion,Iaron," +
			"Illanathaes,Illfanora,Imlarlon,Imyse,Imyvelian,Inferius,Inlurth,innsshe,Iralserin,Irethtalos," +
			"Irholona,Ishal,Ishlashara,Ithelion,Ithlin,Iulil,Jaal,Jamkadi,Kaalume,Kaansera,Karanthanil," +
			"Karnosea,Kasethyr,Keatheas,Kelsya,Keth Aiqua,Kmlon,Kyathlenor,Kyhasera,Lahetheas,Lefdorei," +
			"Lelhamelle,Lilean,Lindeenil,Lindoress,Litys,Llaughei,Lya,Lyfa,Lylharion,Lynathalas,Machei," +
			"Masenoris,Mathethil,Mathentheas,Meethalas,Menyamar,Mithlonde,Mytha,Mythsemelle,Mythsthas,Naahona," +
			"Nalore,Nandeedil,Nasad Ilaurth,Nasin,Nathemar,Neadar,Neilon,Nelalon,Nellean,Nelnetaesi," +
			"Nilenathyr,Nionande,Nylm,Nytenanas,Nythanlenor,O'anlenora,Obeth,Ofaenathyr,Ollmnaes,Ollsmel," +
			"Olwen,Olyaneas,Omanalon,Onelion,Onelond,Orlormel,Ormrion,Oshana,Oshvamel,Raethei,Rauguall," +
			"Reisera,Reslenora,Ryanasera,Rymaserin,Sahnor,Saselune,Sel-Zedraazin,Selananor,Sellerion," +
			"Selmaluma,Shaeras,Shemnas,Shemserin,Sheosari,Sileltalos,Siriande,Siriathil,Srannor,Sshanntyr," +
			"Sshaulu,Syholume,Sylharius,Sylranbel,Taesi,Thalor,Tharenlon,Thelethlune,Thelhohil,Themar,Thene," +
			"Thilfalean,Thilnaenor,Thvethalas,Thylathlond,Tiregul,Tlauven,Tlindhe,Ulal,Ullve,Ulmetheas,Ulssin," +
			"Umnalin,Umye,Umyheserine,Unanneas,Unarith,Undraeth,Unysarion,Vel-Shonidor,Venas,Vin Argor," +
			"Wasrion,Wlalean,Yaeluma,Yeelume,Yethrion,Ymserine,Yueghed,Yuerran,Yuethin",
	},
	{
		// FMG fantasy Dwarven base; dwarf ancestry (FMG base "Dwarven")
		Key:  "dwarven",
		Min:  4,
		Max:  11,
		Dupl: "dk",
		Lexicon: "Addundad,Ahagzad,Ahazil,Akil,Akzizad,Anumush,Araddush,Arar,Arbhur,Badushund,Baragzig,Baragzund," +
			"Barakinb,Barakzig,Barakzinb,Barakzir,Baramunz,Barazinb,Barazir,Bilgabar,Bilgatharb,Bilgathaz," +
			"Bilgila,Bilnaragz,Bilnulbar,Bilnulbun,Bizaddum,Bizaddush,Bizanarg,Bizaram,Bizinbiz,Biziram," +
			"Bunaram,Bundinar,Bundushol,Bundushund,Bundushur,Buzaram,Buzundab,Buzundush,Gabaragz,Gabaram," +
			"Gabilgab,Gabilgath,Gabizir,Gabunal,Gabunul,Gabuzan,Gatharam,Gatharbhur,Gathizdum,Gathuragz," +
			"Gathuraz,Gila,Giledzir,Gilukkhath,Gilukkhel,Gunala,Gunargath,Gunargil,Gundumunz,Gundusharb," +
			"Gundushizd,Kharbharbiln,Kharbhatharb,Kharbhela,Kharbilgab,Kharbuzadd,Khatharbar,Khathizdin," +
			"Khathundush,Khazanar,Khazinbund,Khaziragz,Khaziraz,Khizdabun,Khizdusharbh,Khizdushath,Khizdushel," +
			"Khizdushur,Kholedzar,Khundabiln,Khundabuz,Khundinarg,Khundushel,Khuragzig,Khuramunz,Kibarak," +
			"Kibilnal,Kibizar,Kibunarg,Kibundin,Kibuzan,Kinbadab,Kinbaragz,Kinbarakz,Kinbaram,Kinbizah," +
			"Kinbuzar,Nala,Naledzar,Naledzig,Naledzinb,Naragzah,Naragzar,Naragzig,Narakzah,Narakzar,Naramunz," +
			"Narazar,Nargabad,Nargabar,Nargatharb,Nargila,Nargundum,Nargundush,Nargunul,Narukthar,Narukthel," +
			"Nula,Nulbadush,Nulbaram,Nulbilnarg,Nulbunal,Nulbundab,Nulbundin,Nulbundum,Nulbuzah,Nuledzah," +
			"Nuledzig,Nulukkhaz,Nulukkhund,Nulukkhur,Sharakinb,Sharakzar,Sharamunz,Sharbarukth,Shatharbhizd," +
			"Shatharbiz,Shathazah,Shathizdush,Shathola,Shaziragz,Shizdinar,Shizdushund,Sholukkharb,Shundinulb," +
			"Shundushund,Shurakzund,Shuramunz,Tumunzadd,Tumunzan,Tumunzar,Tumunzinb,Tumunzir,Ukthad,Ulbirad," +
			"Ulbirar,Ulunzar,Ulur,Umunzad,Undalar,Undukkhil,Undun,Undur,Unduzur,Unzar,Unzathun,Usharar," +
			"Zaddinarg,Zaddushur,Zaharbad,Zaharbhizd,Zarakib,Zarakzar,Zaramunz,Zarukthel,Zinbarukth,Zirakinb," +
			"Zirakzir,Ziramunz,Ziruktharbh,Zirukthur,Zundumunz",
	},
	{
		// FMG fantasy Goblin base; goblin ancestry (FMG base "Goblin")
		Key:  "goblin",
		Min:  4,
		Max:  9,
		Dupl: "eag",
		Lexicon: "Asinx,Bhiagielt,Biokvish,Blix,Blus,Bratliaq,Breshass,Bridvelb,Brybsil,Bugbig,Buyagh,Cel,Chalk," +
			"Chiafzia,Chox,Cielb,Cosvil,Crekork,Crild,Croibieq,Diervaq,Dobruing,Driord,Eebligz,Een,Enissee," +
			"Esz,Far,Felhob,Froihiofz,Fruict,Fygsee,Gagablin,Gigganqi,Givzieqee,Glamzofs,Glernaahx,Gneabs," +
			"Gnoklig,Gobbledak,gobbok,Gobbrin,Heszai,Hiszils,Hobgar,Honk,Iahzaarm,Ialsirt,Ilm,Ish,Jasheafta," +
			"Joimtoilm,Kass,Katmelt,Kleabtong,Kleardeek,Klilm,Kluirm,Kuipuinx,Moft,Mogg,Nilbog,Oimzoishai,Onq," +
			"Ozbiard,Paas,Phax,Phigheldai,Preang,Prolkeh,Pyreazzi,Qeerags,Qosx,Rekx,Shaxi,Sios,Slehzit," +
			"Slofboif,Slukex,Srefs,Srurd,Stiaggaltia,Stiolx,Stioskurt,Stroir,Strytzakt,Stuikvact,Styrzangai," +
			"Suirx,Swaxi,Taxai,Thelt,Thresxea,Thult,Traglila,Treaq,Ulb,Ulm,Utha,Utiarm,Veekz,Vohniots," +
			"Vreagaald,Watvielx,Wrogdilk,Wruilt,Xurx,Ziggek,Zriokots",
	},
	{
		// FMG fantasy Orc base; orc, gnoll and hobgoblin ancestries (FMG base "Orc")
		Key:  "orcish",
		Min:  4,
		Max:  8,
		Dupl: "gzrcu",
		Lexicon: "Adgoz,Adgril-Gha,Adog,Adzurd,Agkadh,Agzil-Ghal,Akh,Ariz-Dru,Arkugzo,Arrordri,Ashnedh,Azrurdrekh," +
			"Bagzildre,Bashnud,Bedgez-Graz,Bhakh,Bhegh,Bhiccozdur,Bhicrur,Bhirgoshbel,Bhog,Bhurkrukh," +
			"Bod-Rugniz,Bogzel,Bozdra,Bozgrun,Bozziz,Bral-Lazogh,Brazadh,Brogved,Brogzozir,Brolzug,Brordegeg," +
			"Brorkril-Zrog,Brugroz,Brukh-Zrabrul,Brur-Korre,Bulbredh,Bulgragh,Chaz-Charard,Chegan-Khed,Chugga," +
			"Chuzar,Dhalgron-Mog,Dhazon-Ner,Dhezza,Dhoddud,Dhodh-Brerdrodh,Dhodh-Ghigin,Dhoggun-Bhogh," +
			"Dhulbazzol,Digzagkigh,Dirdrurd,Dodkakh,Dorgri,Drizdedh,Drobagh,Drodh-Ashnugh,Drogvukh-Drodh," +
			"Drukh-Qodgoz,Drurkuz,Dududh,Dur-Khaddol,Egmod,Ekh-Beccon,Ekh-Krerdrugh,Ekh-Mezred,Gagh-Druzred," +
			"Gazdrakh-Vrard,Gegnod,Gerkradh,Ghagrocroz,Ghared-Krin,Ghedgrolbrol,Gheggor,Ghizgil,Gho-Ugnud," +
			"Gholgard,Gidh-Ucceg,Goccogmurd,Golkon,Graz-Khulgag,Gribrabrokh,Gridkog,Grigh-Kaggaz,Grirkrun-Qur," +
			"Grughokh,Grurro,Gugh-Zozgrod,Gur-Ghogkagh,Ibagh-Chol,Ibruzzed,Ibul-Brad,Iggulzaz,Ikh-Ugnan," +
			"Irdrelzug,Irmekh-Bhor,Kacruz,Kalbrugh,Karkor-Zrid,Kazzuz-Zrar,Kezul-Bruz,Kharkiz,Khebun,Khorbric," +
			"Khuldrerra,Khuzdraz,Kirgol,Koggodh,Korkrir-Grar,Kraghird,Krar-Zurmurd,Krigh-Bhurdin,Kroddadh," +
			"Krudh-Khogzokh,Kudgroccukh,Kudrukh,Kudzal,Kuzgrurd-Dedh,Larud,Legvicrodh,Lorgran,Lugekh,Lulkore," +
			"Mazgar,Merkraz,Mocculdrer,Modh-Odod,Morbraz,Mubror,Muccug-Ghuz,Mughakh-Chil,Murmad,Nazad-Ludh," +
			"Negvidh,Nelzor-Zroz,Nirdrukh,Nogvolkar,Nubud,Nuccag,Nudh-Kuldra,Nuzecro,Oddigh-Krodh,Okh-Uggekh," +
			"Ordol,Orkudh-Bhur,Orrad,Qashnagh,Qiccad-Chal,Qiddolzog,Qidzodkakh,Qirzodh,Rarurd,Reradgri,Rezegh," +
			"Rezgrugh,Rodrekh,Rogh-Chirzaz,Rordrushnokh,Rozzez,Ruddirgrad,Rurguz-Vig,Ruzgrin,Ugh-Vruron," +
			"Ughudadh,Uldrukh-Bhudh,Ulgor,Ulkin,Ummugh-Ekh,Uzaggor,Uzdriboz,Uzdroz,Uzord,Uzron,Vaddog," +
			"Vagord-Khod,Velgrudh,Verrugh,Vrazin,Vrobrun,Vrugh-Nardrer,Vrurgu,Vuccidh,Vun-Gaghukh,Zacrad," +
			"Zalbrez,Zigmorbredh,Zordrordud,Zorrudh,Zradgukh,Zragmukh,Zragrizgrakh,Zraldrozzuz,Zrard-Krodog," +
			"Zrazzuz-Vaz,Zrigud,Zrulbukh-Dekh,Zubod-Ur,Zulbriz,Zun-Bergrord",
	},
	{
		// FMG fantasy Draconic base; dragonborn and lizardfolk ancestries (FMG base "Draconic")
		Key:  "draconic",
		Min:  6,
		Max:  14,
		Dupl: "aliuszrox",
		Lexicon: "Aaronarra,Adalon,Adamarondor,Aeglyl,Aerosclughpalar,Aghazstamn,Aglaraerose,Agoshyrvor,Alduin," +
			"Alhazmabad,Altagos,Ammaratha,Amrennathed,Anaglathos,Andrathanach,Araemra,Araugauthos,Arauthator," +
			"Arharzel,Arngalor,Arveiaturace,Athauglas,Augaurath,Auntyrlothtor,Azarvilandral,Azhaq,Balagos," +
			"Baratathlaer,Bleucorundum,BrazzPolis,Canthraxis,Capnolithyl,Charvekkanathor,Chellewis," +
			"Chelnadatilar,Cirrothamalan,Claugiyliamatar,Cragnortherma,Dargentum,Dendeirmerdammarar," +
			"Dheubpurcwenpyl,Domborcojh,Draconobalen,Dragansalor,Dupretiskava,Durnehviir,Eacoathildarandus," +
			"Eldrisithain,Enixtryx,Eormennoth,Esmerandanna,Evenaelorathos,Faenphaele,Felgolos,Felrivenser," +
			"Firkraag,Fll'Yissetat,Furlinastis,Galadaeros,Galglentor,Garnetallisar,Garthammus,Gaulauntyr," +
			"Ghaulantatra,Glouroth,Greshrukk,Guyanothaz,Haerinvureem,Haklashara,Halagaster,Halaglathgar," +
			"Havarlan,Heltipyre,Hethcypressarvil,Hoondarrh,Icehauptannarthanyx,Iiurrendeem,Ileuthra,Iltharagh," +
			"Ingeloakastimizilian,Irdrithkryn,Ishenalyr,Iymrith,Jaerlethket,Jalanvaloss,Jharakkan,Kasidikal," +
			"Kastrandrethilian,Khavalanoth,Khuralosothantar,Kisonraathiisar,Kissethkashaan,Kistarianth,Klauth," +
			"Klithalrundrar,Krashos,Kreston,Kriionfanthicus,Krosulhah,Krustalanos,Kruziikrel,Kuldrak,Lareth," +
			"Latovenomer,Lhammaruntosz,Llimark,Ma'fel'no'sei'kedeh'naar,MaelestorRex,Magarovallanthanz," +
			"Mahatnartorian,Mahrlee,Malaeragoth,Malagarthaul,Malazan,Maldraedior,Maldrithor,MalekSalerno," +
			"Maughrysear,Mejas,Meliordianix,Merah,Mikkaalgensis,Mirmulnir,Mistinarperadnacles,Miteach," +
			"Mithbarazak,Morueme,Moruharzel,Naaslaarum,Nahagliiv,Nalavarauthatoryl,Naxorlytaalsxar,Nevalarich," +
			"Nolalothcaragascint,Nurvureem,Nymmurh,Odahviing,Olothontor,Ormalagos,Otaaryliakkarnos," +
			"Paarthurnax,Pelath,Pelendralaar,Praelorisstan,Praxasalandos,Protanther,Qiminstiir,Quelindritar," +
			"Ralionate,Rathalylaug,Rathguul,Rauglothgor,Raumorthadar,Relonikiv,Ringreemeralxoth,Roraurim," +
			"Rynnarvyx,Sablaxaahl,Sahloknir,Sahrotaar,Samdralyrion,Saryndalaghlothtor,Sawaka,Shalamalauth," +
			"Shammagar,Sharndrel,Shianax,Skarlthoon,Skurge,Smergadas,Ssalangan,Sssurist,Sussethilasis," +
			"Sylvallitham,Tamarand,Tantlevgithus,Tarlacoal,Tenaarlaktor,Thalagyrt,Tharas'kalagram," +
			"Thauglorimorgorus,Thoklastees,Thyka,Tsenshivah,Ueurwen,Uinnessivar,Urnalithorgathla," +
			"Velcuthimmorhar,Velora,Vendrathdammarar,Venomindhar,Viinturuth,Voaraghamanthar,Voslaarum,Vr'tark," +
			"Vrondahorevos,Vuljotnaak,Vulthuryol,Wastirek,Worlathaugh,Xargithorvar,Xavarathimius,Yemere," +
			"Ylithargathril,Ylveraasahlisar,Za-Jikku,Zarlandris,Zellenesterex,Zilanthar,Zormapalearath," +
			"Zundaerazylym,Zz'Pzora",
	},
}
