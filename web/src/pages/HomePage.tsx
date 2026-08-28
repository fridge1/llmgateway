import LandingHeader from "@/components/landing/LandingHeader";
import LandingHero from "@/components/landing/LandingHero";
import StatsCounter from "@/components/landing/StatsCounter";
import ModelMatrix from "@/components/landing/ModelMatrix";
import ModelPricingTable from "@/components/landing/ModelPricingTable";
import FeatureCards from "@/components/landing/FeatureCards";
import PlanCards from "@/components/landing/PlanCards";
import FaqAccordion from "@/components/landing/FaqAccordion";
import LandingCTA from "@/components/landing/LandingCTA";
import LandingFooter from "@/components/landing/LandingFooter";
import { Seo, JsonLd } from "@/components/Seo";
import { ORGANIZATION_JSONLD, WEBSITE_JSONLD } from "@/seo-meta";
import { FAQ_ITEMS } from "@/components/landing/faq-data";

const FAQ_JSONLD = {
  "@context": "https://schema.org",
  "@type": "FAQPage",
  mainEntity: FAQ_ITEMS.map((item) => ({
    "@type": "Question",
    name: item.q,
    acceptedAnswer: { "@type": "Answer", text: item.a },
  })),
};

const HomePage = () => {
  return (
    <div className="w-full min-h-screen bg-background flex flex-col">
      <Seo path="/" />
      <JsonLd data={ORGANIZATION_JSONLD} />
      <JsonLd data={WEBSITE_JSONLD} />
      <JsonLd data={FAQ_JSONLD} />
      <LandingHeader />
      <main className="flex-1">
        <LandingHero />
        <StatsCounter />
        <FeatureCards />
        <ModelMatrix />
        <ModelPricingTable />
        <PlanCards />
        <FaqAccordion />
        <LandingCTA />
      </main>
      <LandingFooter />
    </div>
  );
};

export default HomePage;
